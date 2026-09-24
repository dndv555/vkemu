package web

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"strings"

	"vkemu/internal/model"
	"vkemu/internal/upload"
)

const feedPageSize = 20

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	endTime := atoiOr(r.URL.Query().Get("end_time"), 0)
	posts, err := s.db.FeedRange(v.user.UID, endTime, 0, 0, feedPageSize)

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Новости"))
	body.WriteString(s.postForm())
	body.WriteString(h2("Лента"))
	if err != nil {
		body.WriteString(p("Ошибка загрузки ленты"))
	}
	if len(posts) == 0 {
		body.WriteString(p("Новостей пока нет"))
	}
	for _, post := range posts {
		body.WriteString(s.postBlock(v, post))
	}
	if len(posts) == feedPageSize {
		next := posts[len(posts)-1].Date - 1
		body.WriteString(p(link("/feed?end_time="+strconv.FormatInt(next, 10), "Следующая страница")))
	}
	s.page(w, "Новости", body.String())
}

func (s *Server) postForm() string {
	var body strings.Builder
	body.WriteString(h2("Новая запись"))
	body.WriteString(`<form method="post" action="/post" enctype="multipart/form-data">`)
	body.WriteString(`<p><textarea name="message" rows="3" cols="50"></textarea></p>`)
	body.WriteString(`<p>Вложение (фото, аудио, видео, документ): <input type="file" name="attachment"></p>`)
	body.WriteString(`<p><button type="submit">Отправить</button></p>`)
	body.WriteString(`</form>`)
	return body.String()
}

func (s *Server) postBlock(v viewer, post model.WallPost) string {
	var body strings.Builder
	body.WriteString("<hr>")
	body.WriteString(s.authorBlock(post.FromID, post.Date))
	if post.Text != "" {
		body.WriteString(p(nl2br(html.EscapeString(post.Text))))
	}
	body.WriteString(s.attachmentBlock(post))
	body.WriteString(s.feedLikeBar(v, post))
	body.WriteString(p(link(postURL(post.OwnerID, post.PostID), "Открыть запись и комментарии")))
	return body.String()
}

func (s *Server) feedLikeBar(v viewer, post model.WallPost) string {
	liked := s.db.IsLiked("post", post.OwnerID, post.PostID, v.user.UID)
	count := s.db.LikeCount("post", post.OwnerID, post.PostID)
	comments := s.db.CommentCount("post", post.OwnerID, post.PostID)
	action := "add"
	label := "Нравится"
	if liked {
		action = "delete"
		label = "Убрать лайк"
	}
	var body strings.Builder
	body.WriteString(postButton("/like", []string{
		hidden("owner_id", post.OwnerID),
		hidden("post_id", post.PostID),
		hiddenText("action", action),
		hiddenText("back", "/feed"),
	}, label))
	body.WriteString(p("Лайков: " + strconv.Itoa(count) + ", комментариев: " + strconv.Itoa(comments)))
	return body.String()
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/feed", http.StatusSeeOther)
		return
	}
	parts, err := upload.ReadParts(r)
	if err != nil {
		s.fail(w, "Не удалось прочитать форму")
		return
	}
	message := upload.Field(parts, "message")

	post := model.WallPost{
		OwnerID: v.user.UID,
		FromID:  v.user.UID,
		Date:    s.db.Now(),
		Text:    message,
	}
	if part, ok := upload.File(parts); ok {
		attachType, attachJSON, err := s.uploadAttachment(v.user.UID, part)
		if err != nil {
			s.fail(w, "Не удалось сохранить вложение")
			return
		}
		post.AttachType = attachType
		post.AttachJSON = attachJSON
	}
	if post.Text == "" && post.AttachType == "" {
		s.fail(w, "Запись пустая")
		return
	}
	postID, err := s.db.NextPostID(v.user.UID)
	if err != nil {
		s.fail(w, "Не удалось создать запись")
		return
	}
	post.PostID = postID
	if err := s.db.CreatePost(post); err != nil {
		s.fail(w, "Не удалось сохранить запись")
		return
	}
	http.Redirect(w, r, "/feed", http.StatusSeeOther)
}

func (s *Server) handlePostDelete(w http.ResponseWriter, r *http.Request) {
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
	post, err := s.db.Post(ownerID, postID)
	if err == nil && (post.FromID == v.user.UID || post.OwnerID == v.user.UID) {
		s.db.DeletePost(ownerID, postID)
	}
	http.Redirect(w, r, "/feed", http.StatusSeeOther)
}

func (s *Server) photoAttach(post model.WallPost) string {
	var attach struct {
		Src    string `json:"src"`
		SrcBig string `json:"src_big"`
	}
	if err := json.Unmarshal([]byte(post.AttachJSON), &attach); err != nil {
		return ""
	}
	src := attach.Src
	if src == "" {
		src = attach.SrcBig
	}
	if src == "" {
		return ""
	}
	return p("<img src=\"" + html.EscapeString(src) + "\" width=\"300\" alt=\"photo\">")
}
