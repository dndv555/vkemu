package api

import "vkemu/internal/model"

func (s *Server) newsfeedGet(c *call) (any, error) {
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 20)
	if count <= 0 {
		count = 20
	}
	endTime := c.p.Int("end_time", 0)
	startTime := c.p.Int("start_time", 0)

	posts, err := s.db.FeedRange(c.viewer(), endTime, startTime, offset, count)
	if err != nil {
		return nil, err
	}
	return s.feedResponse(c, posts, false), nil
}

func (s *Server) newsfeedGetComments(c *call) (any, error) {
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 20)
	if count <= 0 {
		count = 20
	}
	endTime := c.p.Int("end_time", 0)
	startTime := c.p.Int("start_time", 0)

	posts, err := s.db.FeedRange(c.viewer(), endTime, startTime, offset, count)
	if err != nil {
		return nil, err
	}
	return s.feedResponse(c, posts, true), nil
}

func (s *Server) feedResponse(c *call, posts []model.WallPost, withComments bool) map[string]any {
	items := make([]any, 0, len(posts))
	userIDs := map[int]bool{c.viewer(): true}
	groupIDs := map[int]bool{}

	for _, post := range posts {
		entry := c.formatPost(post, c.viewer())
		if withComments {
			comments, err := s.db.Comments("post", post.OwnerID, post.PostID, 0, 10)
			if err == nil && len(comments) > 0 {
				commentItems := make([]any, 0, len(comments))
				for _, comment := range comments {
					commentItems = append(commentItems, c.formatComment(comment))
					userIDs[comment.FromID] = true
				}
				entry["comments"] = map[string]any{
					"count":    len(comments),
					"can_post": 1,
					"list":     commentItems,
				}
			}
		}
		items = append(items, entry)
		collectPostAuthors(post, userIDs, groupIDs)
	}

	profiles := make([]any, 0, len(userIDs))
	for uid := range userIDs {
		if user, err := s.db.User(uid); err == nil {
			profiles = append(profiles, c.formatUser(user, "nom"))
		}
	}
	groups := make([]any, 0, len(groupIDs))
	for gid := range groupIDs {
		if list, err := s.db.Groups([]int{gid}); err == nil && len(list) > 0 {
			groups = append(groups, c.formatGroup(list[0]))
		}
	}

	return map[string]any{
		"items":    items,
		"profiles": profiles,
		"groups":   groups,
		"count":    len(items),
	}
}

func collectPostAuthors(post model.WallPost, userIDs, groupIDs map[int]bool) {
	addAuthor(post.FromID, userIDs, groupIDs)
	addAuthor(post.OwnerID, userIDs, groupIDs)
	if post.CopyOwnerID != 0 {
		addAuthor(post.CopyOwnerID, userIDs, groupIDs)
	}
}

func addAuthor(id int, userIDs, groupIDs map[int]bool) {
	switch {
	case id > 0:
		userIDs[id] = true
	case id < 0:
		groupIDs[-id] = true
	}
}
