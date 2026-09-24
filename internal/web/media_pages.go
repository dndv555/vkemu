package web

import (
	"html"
	"net/http"
	"strconv"
	"strings"

	"vkemu/internal/model"
	"vkemu/internal/upload"
)

func (s *Server) handleMusic(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	uid := atoiOr(r.URL.Query().Get("uid"), v.user.UID)
	if uid == 0 {
		uid = v.user.UID
	}

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Музыка: " + s.name(uid)))
	if uid == v.user.UID {
		body.WriteString(s.musicUploadForm())
	}

	items, err := s.db.Audio(uid, 0, 300, 0)
	if err != nil {
		body.WriteString(p("Ошибка загрузки треков"))
	}
	if len(items) == 0 {
		body.WriteString(p("Треков пока нет"))
	}
	for _, item := range items {
		body.WriteString("<hr>")
		body.WriteString(p(html.EscapeString(item.Artist+" — "+item.Title) + " (" + formatDuration(item.Duration) + ")"))
		body.WriteString(p(audioTag(item.URL)))
		if uid == v.user.UID {
			body.WriteString(postButton("/music/delete", []string{hidden("aid", item.AID)}, "Удалить"))
		}
	}
	s.page(w, "Музыка", body.String())
}

func (s *Server) musicUploadForm() string {
	var body strings.Builder
	body.WriteString(h2("Загрузить трек"))
	body.WriteString(`<form method="post" action="/music/upload" enctype="multipart/form-data">`)
	body.WriteString(`<p>Исполнитель: <input type="text" name="artist" value=""></p>`)
	body.WriteString(`<p>Название: <input type="text" name="title" value=""></p>`)
	body.WriteString(`<p>Файл: <input type="file" name="file" accept="audio/*"></p>`)
	body.WriteString(`<p><button type="submit">Загрузить</button></p>`)
	body.WriteString(`</form>`)
	return body.String()
}

func (s *Server) handleMusicUpload(w http.ResponseWriter, r *http.Request) {
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
	artist := strings.TrimSpace(upload.Field(parts, "artist"))
	title := strings.TrimSpace(upload.Field(parts, "title"))
	if _, err := s.saveAudio(v.user.UID, artist, title, part); err != nil {
		s.fail(w, "Не удалось сохранить трек")
		return
	}
	http.Redirect(w, r, "/music", http.StatusSeeOther)
}

func (s *Server) handleMusicDelete(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if aid := formUIDNamed(r, "aid"); aid != 0 {
		s.db.DeleteAudio(v.user.UID, aid)
	}
	http.Redirect(w, r, "/music", http.StatusSeeOther)
}

func (s *Server) handleVideos(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	uid := atoiOr(r.URL.Query().Get("uid"), v.user.UID)
	if uid == 0 {
		uid = v.user.UID
	}

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Видео: " + s.name(uid)))
	if uid == v.user.UID {
		body.WriteString(h2("Загрузить видео"))
		body.WriteString(`<form method="post" action="/videos/upload" enctype="multipart/form-data">`)
		body.WriteString(`<p>Название: <input type="text" name="title" value=""></p>`)
		body.WriteString(`<p>Файл: <input type="file" name="file" accept="video/*"></p>`)
		body.WriteString(`<p><button type="submit">Загрузить</button></p>`)
		body.WriteString(`</form>`)
	}

	videos, err := s.db.Videos(uid, 0, 200)
	if err != nil {
		body.WriteString(p("Ошибка загрузки видео"))
	}
	if len(videos) == 0 {
		body.WriteString(p("Видео пока нет"))
	}
	for _, video := range videos {
		body.WriteString("<hr>")
		body.WriteString(p(html.EscapeString(video.Title)))
		if video.URL != "" {
			body.WriteString(p(videoTag(video.URL)))
		}
	}
	s.page(w, "Видео", body.String())
}

func (s *Server) handleVideoUpload(w http.ResponseWriter, r *http.Request) {
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
	title := strings.TrimSpace(upload.Field(parts, "title"))
	if _, err := s.saveVideo(v.user.UID, title, part); err != nil {
		s.fail(w, "Не удалось сохранить видео")
		return
	}
	http.Redirect(w, r, "/videos", http.StatusSeeOther)
}

func (s *Server) handleNotes(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	uid := atoiOr(r.URL.Query().Get("uid"), v.user.UID)
	if uid == 0 {
		uid = v.user.UID
	}

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Заметки: " + s.name(uid)))
	if uid == v.user.UID {
		body.WriteString(h2("Новая заметка"))
		body.WriteString(`<form method="post" action="/notes/add">`)
		body.WriteString(`<p>Заголовок: <input type="text" name="title" value=""></p>`)
		body.WriteString(`<p><textarea name="text" rows="4" cols="50"></textarea></p>`)
		body.WriteString(`<p><button type="submit">Сохранить</button></p>`)
		body.WriteString(`</form>`)
	}

	notes, err := s.db.Notes(uid, 0, 100)
	if err != nil {
		body.WriteString(p("Ошибка загрузки заметок"))
	}
	if len(notes) == 0 {
		body.WriteString(p("Заметок пока нет"))
	}
	for _, note := range notes {
		body.WriteString("<hr>")
		body.WriteString(h3(note.Title + " — " + formatTime(note.Date)))
		body.WriteString(p(nl2br(html.EscapeString(note.Text))))
		if uid == v.user.UID {
			body.WriteString(postButton("/notes/delete", []string{hidden("nid", note.NID)}, "Удалить"))
		}
	}
	s.page(w, "Заметки", body.String())
}

func (s *Server) handleNoteAdd(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, "Некорректная форма")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	text := strings.TrimSpace(r.FormValue("text"))
	if title == "" && text == "" {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
		return
	}
	if title == "" {
		title = "Без названия"
	}
	nid, err := s.db.NextNoteID()
	if err != nil {
		s.fail(w, "Не удалось сохранить заметку")
		return
	}
	note := model.Note{
		OwnerID: v.user.UID,
		NID:     nid,
		Title:   title,
		Text:    text,
		Date:    s.db.Now(),
	}
	if err := s.db.AddNote(note); err != nil {
		s.fail(w, "Не удалось сохранить заметку")
		return
	}
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

func (s *Server) handleNoteDelete(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if nid := formUIDNamed(r, "nid"); nid != 0 {
		s.db.DeleteNote(v.user.UID, nid)
	}
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	uid := atoiOr(r.URL.Query().Get("uid"), v.user.UID)
	if uid == 0 {
		uid = v.user.UID
	}

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Документы: " + s.name(uid)))
	if uid == v.user.UID {
		body.WriteString(h2("Загрузить документ"))
		body.WriteString(s.uploadForm("/docs/upload", "file", "*/*", "Загрузить", nil))
	}

	docs, err := s.db.Docs(uid, 0, 200)
	if err != nil {
		body.WriteString(p("Ошибка загрузки документов"))
	}
	if len(docs) == 0 {
		body.WriteString(p("Документов пока нет"))
	}
	for _, doc := range docs {
		body.WriteString("<hr>")
		label := doc.Title + " (" + strconv.Itoa(doc.Size) + " байт)"
		body.WriteString(p(link(doc.URL, label)))
		if uid == v.user.UID {
			body.WriteString(postButton("/docs/delete", []string{hidden("did", doc.DID)}, "Удалить"))
		}
	}
	s.page(w, "Документы", body.String())
}

func (s *Server) handleDocUpload(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	part, err := upload.ReadFile(r)
	if err != nil {
		s.fail(w, "Не удалось прочитать файл")
		return
	}
	if _, err := s.saveDoc(v.user.UID, part); err != nil {
		s.fail(w, "Не удалось сохранить документ")
		return
	}
	http.Redirect(w, r, "/docs", http.StatusSeeOther)
}

func (s *Server) handleDocDelete(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if did := formUIDNamed(r, "did"); did != 0 {
		s.db.DeleteDoc(v.user.UID, did)
	}
	http.Redirect(w, r, "/docs", http.StatusSeeOther)
}
