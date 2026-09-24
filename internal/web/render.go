package web

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) page(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, "<!DOCTYPE html>\n<html lang=\"ru\">\n<head>\n<meta charset=\"utf-8\">\n<title>"+
		html.EscapeString(title)+"</title>\n</head>\n<body>\n"+body+"\n</body>\n</html>\n")
}

func (s *Server) fail(w http.ResponseWriter, message string) {
	s.page(w, "Ошибка", h1("Ошибка")+p(html.EscapeString(message))+p(link("/feed", "На главную")))
}

func errorBox(message string) string {
	if message == "" {
		return ""
	}
	return p("<b>" + html.EscapeString(message) + "</b>")
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "0:00"
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func (s *Server) nav(v viewer) string {
	var body strings.Builder
	body.WriteString("<p>")
	body.WriteString(link("/feed", "Новости") + " | ")
	body.WriteString(link(profileURL(v.user.UID), "Моя страница") + " | ")
	body.WriteString(link("/friends", "Друзья") + " | ")
	body.WriteString(link("/messages", "Сообщения") + " | ")
	body.WriteString(link("/photos", "Фото") + " | ")
	body.WriteString(link("/music", "Музыка") + " | ")
	body.WriteString(link("/videos", "Видео") + " | ")
	body.WriteString(link("/notes", "Заметки") + " | ")
	body.WriteString(link("/docs", "Документы") + " | ")
	body.WriteString(link("/places", "Места") + " | ")
	body.WriteString(link("/search", "Поиск") + " | ")
	body.WriteString(link("/settings", "Настройки") + " | ")
	body.WriteString(link("/logout", "Выход"))
	body.WriteString("</p>")
	if v.user.UID != 0 {
		body.WriteString("<p>")
		body.WriteString("<img src=\"" + html.EscapeString(s.userPhoto(v.user.UID)) + "\" width=\"32\" height=\"32\" alt=\"\"> ")
		body.WriteString("<b>" + html.EscapeString(v.user.FullName()) + "</b>")
		body.WriteString("</p>")
	}
	body.WriteString("<hr>")
	return body.String()
}

func (s *Server) name(uid int) string {
	if uid == 0 {
		return "Аноним"
	}
	if user, err := s.db.User(uid); err == nil {
		return user.FullName()
	}
	if uid < 0 {
		return "Сообщество " + strconv.Itoa(-uid)
	}
	return "Пользователь " + strconv.Itoa(uid)
}

func (s *Server) userPhoto(uid int) string {
	if uid < 0 {
		return "/media/photo/" + strconv.Itoa(uid) + "/" + strconv.Itoa(-uid) + ".png"
	}
	if uid > 0 {
		if user, err := s.db.User(uid); err == nil && user.Avatar != "" {
			return user.Avatar
		}
	}
	return "/media/photo/" + strconv.Itoa(uid) + "/" + strconv.Itoa(uid) + ".png"
}

func (s *Server) avatarBlock(v viewer) string {
	var body strings.Builder
	body.WriteString(p("<img src=\"" + html.EscapeString(s.userPhoto(v.user.UID)) + "\" width=\"150\" height=\"150\" alt=\"avatar\">"))
	body.WriteString(s.uploadForm("/upload-avatar", "file", "image/*", "Загрузить аватар", nil))
	return body.String()
}

func nl2br(text string) string {
	return strings.ReplaceAll(text, "\n", "<br>")
}

func atoiOr(value string, def int) int {
	if value == "" {
		return def
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return n
}

func h1(text string) string {
	return "<h1>" + html.EscapeString(text) + "</h1>\n"
}

func h2(text string) string {
	return "<h2>" + html.EscapeString(text) + "</h2>\n"
}

func h3(text string) string {
	return "<h3>" + html.EscapeString(text) + "</h3>\n"
}

func p(inner string) string {
	return "<p>" + inner + "</p>\n"
}

func link(href, label string) string {
	return "<a href=\"" + html.EscapeString(href) + "\">" + html.EscapeString(label) + "</a>"
}

func profileURL(uid int) string {
	return "/id/" + strconv.Itoa(uid)
}

func valueAttr(value string) string {
	return html.EscapeString(value)
}

func (s *Server) uploadForm(action, field, accept, label string, extra map[string]string) string {
	var body strings.Builder
	body.WriteString("<form method=\"post\" action=\"" + html.EscapeString(action) + "\" enctype=\"multipart/form-data\">")
	for name, value := range extra {
		body.WriteString("<input type=\"hidden\" name=\"" + html.EscapeString(name) + "\" value=\"" + valueAttr(value) + "\">")
	}
	body.WriteString("<input type=\"file\" name=\"" + html.EscapeString(field) + "\" accept=\"" + html.EscapeString(accept) + "\"> ")
	body.WriteString("<button type=\"submit\">" + html.EscapeString(label) + "</button>")
	body.WriteString("</form>")
	return body.String()
}

func hidden(name string, value int) string {
	return "<input type=\"hidden\" name=\"" + html.EscapeString(name) + "\" value=\"" + strconv.Itoa(value) + "\">"
}

func hiddenText(name, value string) string {
	return "<input type=\"hidden\" name=\"" + html.EscapeString(name) + "\" value=\"" + valueAttr(value) + "\">"
}

func textInput(name, label, value string) string {
	return "<p>" + html.EscapeString(label) + ": <input type=\"text\" name=\"" + html.EscapeString(name) +
		"\" value=\"" + valueAttr(value) + "\"></p>"
}

func textInputInline(name, value string) string {
	return "<input type=\"text\" name=\"" + html.EscapeString(name) + "\" value=\"" + valueAttr(value) + "\">"
}

type option struct {
	value string
	label string
}

func selectField(name string, options []option, current string) string {
	var body strings.Builder
	body.WriteString("<select name=\"" + html.EscapeString(name) + "\">")
	for _, item := range options {
		selected := ""
		if item.value == current {
			selected = " selected"
		}
		body.WriteString("<option value=\"" + valueAttr(item.value) + "\"" + selected + ">" +
			html.EscapeString(item.label) + "</option>")
	}
	body.WriteString("</select>")
	return body.String()
}

func postButton(action string, fields []string, label string) string {
	var body strings.Builder
	body.WriteString("<form method=\"post\" action=\"" + html.EscapeString(action) + "\">")
	for _, field := range fields {
		body.WriteString(field)
	}
	body.WriteString("<button type=\"submit\">" + html.EscapeString(label) + "</button>")
	body.WriteString("</form>")
	return body.String()
}
