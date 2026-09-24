package web

import (
	"html"
	"net/http"
	"strconv"
	"strings"

	"vkemu/internal/model"
)

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Сообщения"))

	peer := atoiOr(r.URL.Query().Get("uid"), 0)
	chatID := atoiOr(r.URL.Query().Get("chat"), 0)
	if peer == 0 && chatID == 0 {
		body.WriteString(s.dialogsBlock(v))
		s.page(w, "Сообщения", body.String())
		return
	}
	body.WriteString(s.conversationBlock(v, peer, chatID))
	s.page(w, "Сообщения", body.String())
}

func (s *Server) dialogsBlock(v viewer) string {
	dialogs, err := s.db.Dialogs(v.user.UID, 0, 50)
	var body strings.Builder
	body.WriteString(h2("Диалоги"))
	if err != nil {
		body.WriteString(p("Ошибка загрузки диалогов"))
	}
	if len(dialogs) == 0 {
		body.WriteString(p("Диалогов пока нет"))
	}
	for _, msg := range dialogs {
		body.WriteString("<hr>")
		href := "/messages?uid=" + strconv.Itoa(msg.PeerFor(v.user.UID))
		title := s.name(msg.PeerFor(v.user.UID))
		if msg.ChatID != 0 {
			href = "/messages?chat=" + strconv.Itoa(msg.ChatID)
			title = s.db.ChatTitle(msg.ChatID)
		}
		body.WriteString(p("<img src=\"" + html.EscapeString(s.userPhoto(msg.PeerFor(v.user.UID))) +
			"\" width=\"32\" height=\"32\" alt=\"\"> " + link(href, title) + "<br>" + html.EscapeString(msg.Body)))
	}
	body.WriteString(s.startDialogBlock(v))
	return body.String()
}

func (s *Server) startDialogBlock(v viewer) string {
	ids, err := s.db.FriendIDs(v.user.UID)
	var body strings.Builder
	body.WriteString(h2("Написать сообщение"))
	if err != nil || len(ids) == 0 {
		body.WriteString(p("Сначала добавьте друзей на странице " + link("/friends", "Друзья") + "."))
		return body.String()
	}
	options := make([]option, 0, len(ids))
	if users, err := s.db.Users(ids); err == nil {
		for _, user := range users {
			options = append(options, option{value: strconv.Itoa(user.UID), label: user.FullName()})
		}
	}
	body.WriteString(`<form method="get" action="/messages">`)
	body.WriteString("<p>Кому: " + selectField("uid", options, "") + " ")
	body.WriteString("<button type=\"submit\">Открыть диалог</button></p>")
	body.WriteString("</form>")
	return body.String()
}

func (s *Server) conversationBlock(v viewer, peer, chatID int) string {
	var messages []model.Message
	var err error
	title := ""
	if chatID != 0 {
		messages, err = s.db.ChatMessages(chatID, 0, 100)
		title = s.db.ChatTitle(chatID)
	} else {
		messages, err = s.db.Conversation(v.user.UID, peer, 0, 100)
		title = s.name(peer)
	}

	var body strings.Builder
	body.WriteString(p(link("/messages", "← К диалогам")))
	body.WriteString(h2(title))
	if err != nil {
		body.WriteString(p("Ошибка загрузки сообщений"))
	}
	if len(messages) == 0 {
		body.WriteString(p("Сообщений пока нет"))
	}
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		body.WriteString("<hr>")
		body.WriteString(p("<b>" + html.EscapeString(s.name(msg.FromID)) + "</b> — " + formatTime(msg.Date)))
		body.WriteString(p(nl2br(html.EscapeString(msg.Body))))
		body.WriteString(postButton("/messages/delete",
			[]string{hidden("mid", msg.MID), hidden("peer", peer), hidden("chat", chatID)}, "Удалить"))
	}
	body.WriteString(h2("Ответить"))
	body.WriteString(s.messageForm(peer, chatID))
	return body.String()
}

func (s *Server) messageForm(peer, chatID int) string {
	var body strings.Builder
	body.WriteString(`<form method="post" action="/messages/send">`)
	body.WriteString(hidden("uid", peer))
	body.WriteString(hidden("chat_id", chatID))
	body.WriteString(`<p><textarea name="message" rows="2" cols="50"></textarea></p>`)
	body.WriteString(`<p><button type="submit">Отправить</button></p>`)
	body.WriteString(`</form>`)
	return body.String()
}

func (s *Server) handleMessageSend(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, "Некорректная форма")
		return
	}
	peer := atoiOr(r.FormValue("uid"), 0)
	chatID := atoiOr(r.FormValue("chat_id"), 0)
	text := strings.TrimSpace(r.FormValue("message"))
	target := "/messages"
	if chatID != 0 {
		target += "?chat=" + strconv.Itoa(chatID)
	} else if peer != 0 {
		target += "?uid=" + strconv.Itoa(peer)
	}
	if text == "" || (peer == 0 && chatID == 0) {
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}

	toID := peer
	if chatID != 0 {
		toID = 2000000000 + chatID
	}
	msg := model.Message{
		FromID: v.user.UID,
		ToID:   toID,
		ChatID: chatID,
		Date:   s.db.Now(),
		Body:   text,
	}
	if _, err := s.db.AddMessage(msg); err != nil {
		s.fail(w, "Не удалось отправить сообщение")
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) handleMessageDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.require(w, r); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, "Некорректная форма")
		return
	}
	if mid := atoiOr(r.FormValue("mid"), 0); mid != 0 {
		s.db.DeleteMessage(mid)
	}
	peer := atoiOr(r.FormValue("peer"), 0)
	chatID := atoiOr(r.FormValue("chat"), 0)
	target := "/messages"
	if chatID != 0 {
		target += "?chat=" + strconv.Itoa(chatID)
	} else if peer != 0 {
		target += "?uid=" + strconv.Itoa(peer)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
