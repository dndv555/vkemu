package web

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"strings"

	"vkemu/internal/model"
)

func (s *Server) handlePostView(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	ownerID := atoiOr(r.URL.Query().Get("owner"), v.user.UID)
	postID := atoiOr(r.URL.Query().Get("id"), 0)
	if postID == 0 {
		http.Redirect(w, r, "/feed", http.StatusSeeOther)
		return
	}
	post, err := s.db.Post(ownerID, postID)
	if err != nil {
		s.fail(w, "Запись не найдена")
		return
	}

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(s.postDetail(v, post))
	s.page(w, "Запись", body.String())
}

func (s *Server) postDetail(v viewer, post model.WallPost) string {
	var body strings.Builder
	body.WriteString(p(link("/feed", "← К списку записей")))
	body.WriteString("<hr>")
	body.WriteString(s.authorBlock(post.FromID, post.Date))
	if post.Text != "" {
		body.WriteString(p(nl2br(html.EscapeString(post.Text))))
	}
	body.WriteString(s.attachmentBlock(post))
	body.WriteString(s.likeBlock(v, post))
	if post.FromID == v.user.UID || post.OwnerID == v.user.UID {
		body.WriteString(postButton("/post/delete", []string{
			hidden("owner_id", post.OwnerID),
			hidden("post_id", post.PostID),
		}, "Удалить запись"))
	}
	body.WriteString(s.commentsBlock(v, post))
	return body.String()
}

func (s *Server) authorBlock(uid int, date int64) string {
	var body strings.Builder
	body.WriteString(p("<img src=\"" + html.EscapeString(s.userPhoto(uid)) + "\" width=\"40\" height=\"40\" alt=\"photo\"> " +
		"<b>" + html.EscapeString(s.name(uid)) + "</b> — " + formatTime(date)))
	return body.String()
}

func (s *Server) attachmentBlock(post model.WallPost) string {
	switch post.AttachType {
	case "photo":
		return s.photoAttach(post)
	case "audio":
		var attach struct {
			Artist string `json:"artist"`
			Title  string `json:"title"`
			URL    string `json:"url"`
		}
		if json.Unmarshal([]byte(post.AttachJSON), &attach) != nil {
			return ""
		}
		return p(html.EscapeString(attach.Artist+" — "+attach.Title) + "<br>" + audioTag(attach.URL))
	case "video":
		var attach struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		}
		if json.Unmarshal([]byte(post.AttachJSON), &attach) != nil {
			return ""
		}
		if attach.URL != "" {
			return p(videoTag(attach.URL))
		}
		return p("[видео] " + html.EscapeString(attach.Title))
	case "doc":
		return s.docAttach(post)
	case "link":
		var attach struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		}
		if json.Unmarshal([]byte(post.AttachJSON), &attach) != nil {
			return ""
		}
		return p("[ссылка] " + link(attach.URL, attach.Title))
	case "note":
		var attach struct {
			Title string `json:"title"`
			Text  string `json:"text"`
		}
		if json.Unmarshal([]byte(post.AttachJSON), &attach) != nil {
			return ""
		}
		return p("[заметка] <b>" + html.EscapeString(attach.Title) + "</b><br>" + nl2br(html.EscapeString(attach.Text)))
	case "geo":
		return s.geoAttach(post)
	}
	return ""
}

func (s *Server) docAttach(post model.WallPost) string {
	var attach struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	if json.Unmarshal([]byte(post.AttachJSON), &attach) != nil {
		return ""
	}
	return p("[документ] " + link(attach.URL, attach.Title))
}

func (s *Server) geoAttach(post model.WallPost) string {
	var geo struct {
		Place struct {
			PlaceID int    `json:"place_id"`
			Title   string `json:"title"`
		} `json:"place"`
	}
	if json.Unmarshal([]byte(post.GeoJSON), &geo) != nil || geo.Place.Title == "" {
		return ""
	}
	return p("[место] " + link("/places", geo.Place.Title))
}

func (s *Server) likeBlock(v viewer, post model.WallPost) string {
	liked := s.db.IsLiked("post", post.OwnerID, post.PostID, v.user.UID)
	count := s.db.LikeCount("post", post.OwnerID, post.PostID)
	action := "add"
	label := "Нравится (" + strconv.Itoa(count) + ")"
	if liked {
		action = "delete"
		label = "Убрать лайк (" + strconv.Itoa(count) + ")"
	}
	return postButton("/like", []string{
		hidden("owner_id", post.OwnerID),
		hidden("post_id", post.PostID),
		hiddenText("action", action),
	}, label)
}

func (s *Server) handleLike(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/feed", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, "Некорректная форма")
		return
	}
	ownerID := atoiOr(r.FormValue("owner_id"), 0)
	postID := atoiOr(r.FormValue("post_id"), 0)
	if r.FormValue("action") == "delete" {
		s.db.Unlike("post", ownerID, postID, v.user.UID)
	} else {
		s.db.Like("post", ownerID, postID, v.user.UID)
	}
	s.refreshPostCounters(ownerID, postID)
	s.redirectBack(w, r, postURL(ownerID, postID))
}

func (s *Server) commentsBlock(v viewer, post model.WallPost) string {
	var body strings.Builder
	body.WriteString(h2("Комментарии"))
	comments, err := s.db.Comments("post", post.OwnerID, post.PostID, 0, 100)
	if err != nil {
		body.WriteString(p("Ошибка загрузки комментариев"))
	}
	if len(comments) == 0 {
		body.WriteString(p("Комментариев пока нет"))
	}
	for _, comment := range comments {
		body.WriteString("<hr>")
		body.WriteString(s.authorBlock(comment.FromID, comment.Date))
		body.WriteString(p(nl2br(html.EscapeString(comment.Text))))
	}
	body.WriteString(h3("Оставить комментарий"))
	body.WriteString(`<form method="post" action="/comment">`)
	body.WriteString(hidden("owner_id", post.OwnerID))
	body.WriteString(hidden("post_id", post.PostID))
	body.WriteString(`<p><textarea name="text" rows="2" cols="50"></textarea></p>`)
	body.WriteString(`<p><button type="submit">Отправить</button></p>`)
	body.WriteString("</form>")
	return body.String()
}

func (s *Server) handleComment(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/feed", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, "Некорректная форма")
		return
	}
	ownerID := atoiOr(r.FormValue("owner_id"), 0)
	postID := atoiOr(r.FormValue("post_id"), 0)
	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		http.Redirect(w, r, postURL(ownerID, postID), http.StatusSeeOther)
		return
	}
	comment := model.Comment{
		CType:   "post",
		OwnerID: ownerID,
		ItemID:  postID,
		FromID:  v.user.UID,
		Date:    s.db.Now(),
		Text:    text,
	}
	if err := s.db.AddComment(comment); err != nil {
		s.fail(w, "Не удалось сохранить комментарий")
		return
	}
	s.refreshPostCounters(ownerID, postID)
	s.redirectBack(w, r, postURL(ownerID, postID))
}

func (s *Server) refreshPostCounters(ownerID, postID int) {
	if _, err := s.db.Post(ownerID, postID); err != nil {
		return
	}
	likes := s.db.LikeCount("post", ownerID, postID)
	comments := s.db.CommentCount("post", ownerID, postID)
	s.db.SetPostCounters(ownerID, postID, likes, comments)
}

func postURL(ownerID, postID int) string {
	return "/post/view?owner=" + strconv.Itoa(ownerID) + "&id=" + strconv.Itoa(postID)
}

func audioTag(src string) string {
	if src == "" {
		return ""
	}
	return "<audio controls preload=\"none\" src=\"" + html.EscapeString(src) + "\"></audio>"
}

func videoTag(src string) string {
	if src == "" {
		return ""
	}
	return "<video controls preload=\"none\" width=\"400\" src=\"" + html.EscapeString(src) + "\"></video>"
}
