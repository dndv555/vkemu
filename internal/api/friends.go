package api

import (
	"vkemu/internal/model"
)

func (s *Server) friendsGet(c *call) (any, error) {
	uid := c.p.Int("uid", c.viewer())
	if uid == 0 {
		uid = c.viewer()
	}
	ids, err := s.db.FriendIDs(uid)
	if err != nil {
		return nil, err
	}
	users, err := s.db.Users(ids)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(users))
	for _, user := range users {
		out = append(out, c.formatUser(user, "nom"))
	}
	return out, nil
}

func (s *Server) friendsGetOnline(c *call) (any, error) {
	ids, err := s.db.OnlineFriendIDs(c.viewer())
	if err != nil {
		return nil, err
	}
	return intSliceToAny(ids), nil
}

func (s *Server) friendsGetMutual(c *call) (any, error) {
	target := c.p.Int("target_uid", 0)
	ids, err := s.db.MutualFriendIDs(c.viewer(), target)
	if err != nil {
		return nil, err
	}
	return intSliceToAny(ids), nil
}

func (s *Server) friendsGetRequests(c *call) (any, error) {
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 20)
	if count <= 0 {
		count = 20
	}
	requests, err := s.db.FriendRequests(c.viewer(), offset, count)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(requests))
	for _, request := range requests {
		entry := map[string]any{
			"uid":     request.FromUID,
			"message": request.Message,
			"date":    int(request.Date),
		}
		if c.p.Bool("need_mutual", false) {
			mutual, err := s.db.MutualFriendIDs(c.viewer(), request.FromUID)
			if err != nil {
				return nil, err
			}
			limit := len(mutual)
			if limit > 5 {
				limit = 5
			}
			entry["mutual"] = map[string]any{
				"count": len(mutual),
				"users": intSliceToAny(mutual[:limit]),
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *Server) friendsAdd(c *call) (any, error) {
	uid := c.p.Int("uid", 0)
	if uid == 0 {
		return nil, errParam
	}
	text := c.p.String("text", "")
	viewer := c.viewer()

	if s.db.AreFriends(viewer, uid) {
		return 4, nil
	}
	if _, err := s.db.FriendRequest(uid, viewer); err == nil {
		s.db.UpsertFriend(viewer, uid)
		s.db.UpsertFriend(uid, viewer)
		s.db.DeleteFriendRequest(uid, viewer)
		s.srvPush(uid, []any{8, -viewer})
		return 2, nil
	}
	if err := s.db.SaveFriendRequest(model.FriendRequest{
		FromUID: viewer,
		ToUID:   uid,
		Message: text,
		Date:    s.db.Now(),
	}); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) friendsDelete(c *call) (any, error) {
	uid := c.p.Int("uid", 0)
	if uid == 0 {
		return nil, errParam
	}
	if err := s.db.DeleteFriend(c.viewer(), uid); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) friendsGetByPhones(c *call) (any, error) {
	phones := c.p.List("phones")
	out := make([]any, 0, len(phones))
	for _, phone := range phones {
		user, err := s.db.UserByDomain(phone)
		if err != nil {
			continue
		}
		out = append(out, c.formatUser(user, "nom"))
	}
	return out, nil
}

func (s *Server) friendsAreFriends(c *call) (any, error) {
	uids := c.p.IntList("uids")
	out := make([]any, 0, len(uids))
	for _, uid := range uids {
		out = append(out, map[string]any{
			"uid":           uid,
			"friend_status": boolInt(s.db.AreFriends(c.viewer(), uid)) * 3,
		})
	}
	return out, nil
}

func (s *Server) srvPush(uid int, update []any) {
	s.hub.Push(uid, update...)
}
