package api

import (
	"sort"
	"strconv"

	"vkemu/internal/model"
)

func (s *Server) usersGet(c *call) (any, error) {
	explicit := c.p.Has("uids")
	uids := c.p.IntList("uids")
	if len(uids) == 0 {
		if uid := c.p.Int("uid", 0); uid != 0 {
			uids = append(uids, uid)
			explicit = false
		}
	}
	if len(uids) == 0 {
		if explicit {
			return []any{}, nil
		}
		uids = append(uids, c.viewer())
	}
	nameCase := c.p.String("name_case", "nom")

	users, err := s.db.Users(uids)
	if err != nil {
		return nil, err
	}
	byUID := make(map[int]model.User, len(users))
	for _, user := range users {
		byUID[user.UID] = user
	}
	out := make([]any, 0, len(uids))
	for _, uid := range uids {
		if uid == 0 {
			continue
		}
		if user, ok := byUID[uid]; ok {
			out = append(out, c.formatUser(user, nameCase))
			continue
		}
		if uid < 0 {
			continue
		}
		out = append(out, deletedUser(uid))
	}
	return out, nil
}

func (s *Server) usersSearch(c *call) (any, error) {
	query := c.p.String("q", "")
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 20)
	if count <= 0 {
		count = 20
	}

	exclude := []int{c.viewer()}
	users, err := s.db.SearchUsers(query, exclude)
	if err != nil {
		return nil, err
	}
	if offset > len(users) {
		offset = len(users)
	}
	end := offset + count
	if end > len(users) {
		end = len(users)
	}
	page := users[offset:end]

	out := make([]any, 0, len(page)+1)
	out = append(out, len(users))
	for _, user := range page {
		out = append(out, c.formatUser(user, "nom"))
	}
	return out, nil
}

func (s *Server) usersGetSubscriptions(c *call) (any, error) {
	uid := c.p.Int("uid", c.viewer())
	friends, err := s.db.FriendIDs(uid)
	if err != nil {
		return nil, err
	}
	sort.Ints(friends)
	return map[string]any{
		"users":  map[string]any{"count": len(friends), "items": intSliceToAny(friends)},
		"groups": map[string]any{"count": 0, "items": []any{}},
	}, nil
}

func deletedUser(uid int) map[string]any {
	return map[string]any{
		"uid":        uid,
		"first_name": "DELETED",
		"last_name":  "",
		"photo":      "",
		"photo_rec":  "",
		"online":     0,
	}
}

func intSliceToAny(values []int) []any {
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return out
}

func parseUID(value string) (int, bool) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return n, true
}
