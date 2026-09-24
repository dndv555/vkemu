package api

import (
	"strconv"

	"vkemu/internal/model"
	"vkemu/internal/params"
)

func (s *Server) messagesGetDialogs(c *call) (any, error) {
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 20)
	if count <= 0 {
		count = 20
	}
	dialogs, err := s.db.Dialogs(c.viewer(), offset, count)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(dialogs)+1)
	out = append(out, s.db.DialogCount(c.viewer()))
	for _, msg := range dialogs {
		out = append(out, c.formatMessage(msg, c.viewer()))
	}
	return out, nil
}

func (s *Server) messagesGetHistory(c *call) (any, error) {
	uid := c.p.Int("uid", 0)
	chatID := c.p.Int("chat_id", 0)
	if uid == 0 && chatID == 0 {
		return nil, errParam
	}
	peer := uid
	if chatID != 0 {
		peer = chatID
	}
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 20)
	if count <= 0 {
		count = 20
	}

	var messages []model.Message
	var err error
	if chatID != 0 {
		messages, err = s.chatHistory(chatID, offset, count)
	} else {
		messages, err = s.db.Messages(peer, offset, count)
	}
	if err != nil {
		return nil, err
	}
	total := len(messages)
	out := make([]any, 0, len(messages)+1)
	out = append(out, total)
	for _, msg := range messages {
		out = append(out, c.formatMessage(msg, c.viewer()))
	}
	return out, nil
}

func (s *Server) chatHistory(chatID, offset, count int) ([]model.Message, error) {
	return s.db.ChatMessages(chatID, offset, count)
}

func (s *Server) messagesGetByID(c *call) (any, error) {
	ids := c.p.IntList("mid")
	found, err := s.db.MessagesByID(ids)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(found)+1)
	out = append(out, len(found))
	for _, msg := range found {
		out = append(out, c.formatMessage(msg, c.viewer()))
	}
	return out, nil
}

func (s *Server) messagesGet(c *call) (any, error) {
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 20)
	if count <= 0 {
		count = 20
	}
	dialogs, err := s.db.Dialogs(c.viewer(), offset, count)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(dialogs)+1)
	out = append(out, s.db.DialogCount(c.viewer()))
	for _, msg := range dialogs {
		out = append(out, c.formatMessage(msg, c.viewer()))
	}
	return out, nil
}

func (s *Server) messagesSend(c *call) (any, error) {
	uid := c.p.Int("uid", 0)
	chatID := c.p.Int("chat_id", 0)
	if uid == 0 && chatID == 0 {
		return nil, errParam
	}
	body := c.p.String("message", "")
	if body == "" {
		return nil, errf(100, "Message is empty.")
	}

	toID := uid
	if chatID != 0 {
		toID = 2000000000 + chatID
	}
	msg := model.Message{
		FromID: c.viewer(),
		ToID:   toID,
		ChatID: chatID,
		Date:   s.db.Now(),
		Body:   body,
	}
	mid, err := s.db.AddMessage(msg)
	if err != nil {
		return nil, err
	}
	msg.MID = mid

	if chatID == 0 {
		s.hub.Push(uid, []any{4, mid, 1, c.viewer(), int(msg.Date), "", body, ""}...)
		s.hub.Push(c.viewer(), []any{4, mid, 2, uid, int(msg.Date), "", body, ""}...)
	} else {
		members, _ := s.db.ChatMembers(chatID)
		targets := map[int]bool{c.viewer(): true}
		for _, member := range members {
			targets[member] = true
		}
		for member := range targets {
			flags := 1
			if member == c.viewer() {
				flags = 2
			}
			s.hub.Push(member, []any{4, mid, flags, toID, int(msg.Date), "", body, ""}...)
		}
	}
	return mid, nil
}

func (s *Server) messagesDelete(c *call) (any, error) {
	ids := c.p.IntList("mid")
	for _, mid := range ids {
		if err := s.db.DeleteMessage(mid); err != nil {
			return nil, err
		}
	}
	return 1, nil
}

func (s *Server) messagesMarkAsRead(c *call) (any, error) {
	ids := c.p.IntList("mids")
	if err := s.db.MarkMessagesRead(ids, c.viewer()); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) messagesSearch(c *call) (any, error) {
	query := c.p.String("q", "")
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 20)
	if count <= 0 {
		count = 20
	}
	found, err := s.db.SearchMessages(c.viewer(), query, offset, count)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(found)+1)
	out = append(out, len(found))
	for _, msg := range found {
		out = append(out, c.formatMessage(msg, c.viewer()))
	}
	return out, nil
}

func (s *Server) messagesGetLongPollServer(c *call) (any, error) {
	key := params.MD5("lp:" + strconv.Itoa(c.viewer()) + ":" + strconv.FormatInt(s.db.Now(), 10))
	s.hub.BindKey(key, c.viewer())
	host := c.srv.longPollHost(c)
	return map[string]any{
		"server": host,
		"key":    key,
		"ts":     s.hub.TS(c.viewer()),
	}, nil
}

func (s *Server) getCounters(c *call) (any, error) {
	counters := map[string]any{
		"messages":      s.db.UnreadCount(c.viewer()),
		"friends":       s.db.FriendRequestCount(c.viewer()),
		"photos":        0,
		"events":        0,
		"groups":        0,
		"notifications": 0,
	}
	return counters, nil
}

func (s *Server) activityOnline(c *call) (any, error) {
	s.db.SetUserOnline(c.viewer(), true)
	return 1, nil
}
