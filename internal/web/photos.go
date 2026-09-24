package web

import (
	"html"
	"net/http"
	"strconv"
	"strings"

	"vkemu/internal/model"
	"vkemu/internal/upload"
)

func (s *Server) handlePhotos(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	aid := atoiOr(r.URL.Query().Get("aid"), 0)

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Фото"))
	body.WriteString(s.photoUploadForm())

	albums, _ := s.db.Albums(v.user.UID)
	body.WriteString(h2("Альбомы"))
	body.WriteString(p(link("/photos", "Все фото")))
	for _, album := range albums {
		label := album.Title + " (" + strconv.Itoa(s.db.PhotoCount(album.OwnerID, album.AID)) + ")"
		body.WriteString(p(link("/photos?aid="+strconv.Itoa(album.AID), label)))
	}
	body.WriteString(h2("Новый альбом"))
	body.WriteString(`<form method="post" action="/albums/create">`)
	body.WriteString(`<p>Название: <input type="text" name="title" value=""></p>`)
	body.WriteString(`<p><button type="submit">Создать</button></p>`)
	body.WriteString(`</form>`)

	body.WriteString(h2("Снимки"))
	photos, err := s.db.Photos(v.user.UID, aid, 0, 200)
	if err != nil {
		body.WriteString(p("Ошибка загрузки фото"))
	}
	if len(photos) == 0 {
		body.WriteString(p("Фотографий пока нет"))
	}
	for _, photo := range photos {
		src := photo.Src
		if src == "" {
			src = photo.SrcBig
		}
		body.WriteString("<hr>")
		body.WriteString(p("<img src=\"" + html.EscapeString(src) + "\" width=\"220\" alt=\"photo\">"))
		if photo.Text != "" {
			body.WriteString(p(html.EscapeString(photo.Text)))
		}
		body.WriteString(postButton("/photos/delete", []string{hidden("pid", photo.PID)}, "Удалить"))
	}
	s.page(w, "Фото", body.String())
}

func (s *Server) photoUploadForm() string {
	var body strings.Builder
	body.WriteString(h2("Загрузить фото"))
	body.WriteString(`<form method="post" action="/photos/upload" enctype="multipart/form-data">`)
	body.WriteString(`<p>Подпись: <input type="text" name="caption" value=""></p>`)
	body.WriteString(`<p>Файл: <input type="file" name="photo" accept="image/*"></p>`)
	body.WriteString(`<p><button type="submit">Загрузить</button></p>`)
	body.WriteString(`</form>`)
	return body.String()
}

func (s *Server) handlePhotoUpload(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	parts, err := upload.ReadParts(r)
	if err != nil {
		s.fail(w, "Не удалось прочитать форму")
		return
	}
	part, ok := upload.File(parts)
	if !ok {
		s.fail(w, "Файл не выбран")
		return
	}
	photo, err := s.savePhoto(v.user.UID, part)
	if err != nil {
		s.fail(w, "Не удалось сохранить фото")
		return
	}
	if caption := strings.TrimSpace(upload.Field(parts, "caption")); caption != "" {
		s.db.SetPhotoText(v.user.UID, photo.PID, caption)
	}
	http.Redirect(w, r, "/photos", http.StatusSeeOther)
}

func (s *Server) handlePhotoDelete(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if pid := formUIDNamed(r, "pid"); pid != 0 {
		s.db.DeletePhoto(v.user.UID, pid)
	}
	http.Redirect(w, r, "/photos", http.StatusSeeOther)
}

func (s *Server) handleAlbumCreate(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, "Некорректная форма")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Redirect(w, r, "/photos", http.StatusSeeOther)
		return
	}
	aid, err := s.db.NextAlbumID()
	if err != nil {
		s.fail(w, "Не удалось создать альбом")
		return
	}
	album := model.Album{
		OwnerID: v.user.UID,
		AID:     aid,
		Title:   title,
		Privacy: 0,
		ThumbID: -1,
		Created: s.db.Now(),
	}
	if err := s.db.AddAlbum(album); err != nil {
		s.fail(w, "Не удалось создать альбом")
		return
	}
	http.Redirect(w, r, "/photos?aid="+strconv.Itoa(aid), http.StatusSeeOther)
}

func formUIDNamed(r *http.Request, name string) int {
	if err := r.ParseForm(); err != nil {
		return 0
	}
	return atoiOr(r.FormValue(name), 0)
}
