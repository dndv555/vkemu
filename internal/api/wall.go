package api

import (
	"encoding/json"
	"strconv"
	"strings"

	"vkemu/internal/model"
)

func (s *Server) wallGet(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	if ownerID == 0 {
		ownerID = c.viewer()
	}
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 10)
	if count <= 0 {
		count = 10
	}

	posts, err := s.db.Posts(ownerID, offset, count)
	if err != nil {
		return nil, err
	}
	total := s.db.PostCount(ownerID)

	out := make([]any, 0, len(posts)+1)
	out = append(out, total)
	for _, post := range posts {
		out = append(out, c.formatPost(post, c.viewer()))
	}
	return out, nil
}

func (s *Server) wallGetByID(c *call) (any, error) {
	posts := c.p.List("posts")
	ids := make([]int, 0, len(posts))
	for _, item := range posts {
		if idx := strings.Index(item, "_"); idx >= 0 {
			if id, ok := parseUID(item[idx+1:]); ok {
				ids = append(ids, id)
			}
			continue
		}
		if id, ok := parseUID(item); ok {
			ids = append(ids, id)
		}
	}
	found, err := s.db.PostsByID(ids)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(found))
	for _, post := range found {
		out = append(out, c.formatPost(post, c.viewer()))
	}
	return out, nil
}

func (s *Server) wallPost(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	postID, err := s.db.NextPostID(ownerID)
	if err != nil {
		return nil, err
	}
	post := model.WallPost{
		OwnerID: ownerID,
		PostID:  postID,
		FromID:  c.viewer(),
		Date:    s.db.Now(),
		Text:    c.p.String("message", ""),
	}

	if attachment := c.p.String("attachment", ""); attachment != "" {
		s.applyAttachment(&post, attachment)
	}
	if place := c.p.Int("place_id", 0); place != 0 {
		post.GeoJSON = `{"place":{"place_id":` + strconv.Itoa(place) + `}}`
	}
	if err := s.db.CreatePost(post); err != nil {
		return nil, err
	}

	s.hub.Push(ownerID, []any{1, postID, 0}...)
	return map[string]any{"post_id": postID}, nil
}

func (s *Server) applyAttachment(post *model.WallPost, attachment string) {
	parts := strings.SplitN(attachment, ",", 2)
	item := parts[0]
	switch {
	case strings.HasPrefix(item, "photo"):
		ids := strings.Split(strings.TrimPrefix(item, "photo"), "_")
		if len(ids) != 2 {
			return
		}
		ownerID, _ := parseUID(ids[0])
		pid, _ := parseUID(ids[1])
		photos, err := s.db.PhotoByID([][2]int{{ownerID, pid}})
		if err != nil || len(photos) == 0 {
			return
		}
		post.AttachType = "photo"
		post.AttachJSON = photoAttachJSON(photos[0], false)
	case strings.HasPrefix(item, "audio"):
		ids := strings.Split(strings.TrimPrefix(item, "audio"), "_")
		if len(ids) != 2 {
			return
		}
		ownerID, _ := parseUID(ids[0])
		aid, _ := parseUID(ids[1])
		audios, err := s.db.AudioByID([][2]int{{ownerID, aid}})
		if err != nil || len(audios) == 0 {
			return
		}
		post.AttachType = "audio"
		post.AttachJSON = audioAttachJSON(audios[0])
	case strings.HasPrefix(item, "video"):
		ids := strings.Split(strings.TrimPrefix(item, "video"), "_")
		if len(ids) != 2 {
			return
		}
		ownerID, _ := parseUID(ids[0])
		vid, _ := parseUID(ids[1])
		video, err := s.db.Video(ownerID, vid)
		if err != nil {
			return
		}
		post.AttachType = "video"
		post.AttachJSON = videoAttachJSON(video)
	case strings.HasPrefix(item, "note"):
		ids := strings.Split(strings.TrimPrefix(item, "note"), "_")
		if len(ids) != 2 {
			return
		}
		ownerID, _ := parseUID(ids[0])
		nid, _ := parseUID(ids[1])
		note, err := s.db.Note(ownerID, nid)
		if err != nil {
			return
		}
		post.AttachType = "note"
		post.AttachJSON = noteAttachJSON(note)
	case strings.HasPrefix(item, "doc"):
		ids := strings.Split(strings.TrimPrefix(item, "doc"), "_")
		if len(ids) != 2 {
			return
		}
		ownerID, _ := parseUID(ids[0])
		did, _ := parseUID(ids[1])
		doc, err := s.db.Doc(ownerID, did)
		if err != nil {
			return
		}
		post.AttachType = "doc"
		post.AttachJSON = docAttachJSON(doc)
	case strings.HasPrefix(item, "poll"):
		ids := strings.Split(strings.TrimPrefix(item, "poll"), "_")
		if len(ids) != 2 {
			return
		}
		ownerID, _ := parseUID(ids[0])
		pollID, _ := parseUID(ids[1])
		poll, err := s.db.Poll(ownerID, pollID)
		if err != nil {
			return
		}
		post.AttachType = "poll"
		post.AttachJSON = pollAttachJSON(poll)
	case strings.HasPrefix(item, "http://") || strings.HasPrefix(item, "https://"):
		host := item[strings.Index(item, "://")+3:]
		if idx := strings.Index(host, "/"); idx >= 0 {
			host = host[:idx]
		}
		post.AttachType = "link"
		post.AttachJSON = linkAttachJSON(item, host)
	}
}

func (s *Server) wallDelete(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	postID := c.p.Int("post_id", 0)
	if postID == 0 {
		return nil, errParam
	}
	if err := s.db.DeletePost(ownerID, postID); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) wallEdit(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	postID := c.p.Int("post_id", 0)
	if postID == 0 {
		return nil, errParam
	}
	post, err := s.db.Post(ownerID, postID)
	if err != nil {
		return nil, errNotFound
	}
	post.Text = c.p.String("message", post.Text)
	if attachment := c.p.String("attachment", ""); attachment != "" {
		post.AttachType = ""
		post.AttachJSON = ""
		s.applyAttachment(&post, attachment)
	}
	if err := s.db.UpdatePost(post); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) wallGetComments(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	postID := c.p.Int("post_id", 0)
	return s.comments(c, "post", ownerID, postID)
}

func (s *Server) photosGetComments(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	pid := c.p.Int("pid", 0)
	return s.comments(c, "photo", ownerID, pid)
}

func (s *Server) videoGetComments(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	vid := c.p.Int("vid", 0)
	return s.comments(c, "video", ownerID, vid)
}

func (s *Server) notesGetComments(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	nid := c.p.Int("nid", 0)
	return s.comments(c, "note", ownerID, nid)
}

func (s *Server) comments(c *call, cType string, ownerID, itemID int) (any, error) {
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 100)
	if count <= 0 {
		count = 100
	}
	list, err := s.db.Comments(cType, ownerID, itemID, offset, count)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(list)+1)
	out = append(out, s.db.CommentCount(cType, ownerID, itemID))
	for _, comment := range list {
		out = append(out, c.formatComment(comment))
	}
	return out, nil
}

func (s *Server) wallAddComment(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	postID := c.p.Int("post_id", 0)
	comment := model.Comment{
		CType:      "post",
		OwnerID:    ownerID,
		ItemID:     postID,
		FromID:     c.viewer(),
		Date:       s.db.Now(),
		Text:       c.p.String("text", ""),
		ReplyToUID: c.p.Int("reply_to_uid", 0),
	}
	if err := s.db.AddComment(comment); err != nil {
		return nil, err
	}
	comments := s.db.CommentCount("post", ownerID, postID)
	post, err := s.db.Post(ownerID, postID)
	if err == nil {
		s.db.SetPostCounters(ownerID, postID, post.Likes, comments)
	}
	return comments, nil
}

func (s *Server) wallDeleteComment(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	cid := c.p.Int("cid", 0)
	postID := c.p.Int("post_id", 0)
	if err := s.db.DeleteComment("post", ownerID, postID, cid); err != nil {
		return nil, err
	}
	comments := s.db.CommentCount("post", ownerID, postID)
	if post, err := s.db.Post(ownerID, postID); err == nil {
		s.db.SetPostCounters(ownerID, postID, post.Likes, comments)
	}
	return 1, nil
}

func (s *Server) wallAddLike(c *call) (any, error) {
	return s.likeToggle(c, true)
}

func (s *Server) wallDeleteLike(c *call) (any, error) {
	return s.likeToggle(c, false)
}

func (s *Server) likeToggle(c *call, add bool) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	postID := c.p.Int("post_id", 0)
	if add {
		_, likes := s.db.Like("post", ownerID, postID, c.viewer())
		return map[string]any{"likes": likes}, nil
	}
	_, likes := s.db.Unlike("post", ownerID, postID, c.viewer())
	return map[string]any{"likes": likes}, nil
}

func (s *Server) likesAdd(c *call) (any, error) {
	return s.likesGeneric(c, true)
}

func (s *Server) likesDelete(c *call) (any, error) {
	return s.likesGeneric(c, false)
}

func (s *Server) likesGeneric(c *call, add bool) (any, error) {
	cType := c.p.String("type", "post")
	ownerID := c.p.Int("owner_id", 0)
	itemID := c.p.Int("item_id", 0)
	if add {
		_, likes := s.db.Like(cType, ownerID, itemID, c.viewer())
		return map[string]any{"likes": likes}, nil
	}
	_, likes := s.db.Unlike(cType, ownerID, itemID, c.viewer())
	return map[string]any{"likes": likes}, nil
}

func photoAttachJSON(photo model.Photo, big bool) string {
	src := photo.Src
	body := map[string]any{
		"owner_id": photo.OwnerID,
		"pid":      photo.PID,
		"aid":      photo.AID,
		"src":      src,
		"src_big":  photo.SrcBig,
		"text":     photo.Text,
	}
	raw, _ := json.Marshal(body)
	return string(raw)
}

func audioAttachJSON(item model.Audio) string {
	body := map[string]any{
		"owner_id": item.OwnerID,
		"aid":      item.AID,
		"artist":   item.Artist,
		"title":    item.Title,
		"duration": item.Duration,
	}
	raw, _ := json.Marshal(body)
	return string(raw)
}

func videoAttachJSON(video model.Video) string {
	body := map[string]any{
		"owner_id": video.OwnerID,
		"vid":      video.VID,
		"title":    video.Title,
		"image":    video.Image,
	}
	raw, _ := json.Marshal(body)
	return string(raw)
}

func pollAttachJSON(poll model.Poll) string {
	answers := make([]any, 0, len(poll.Answers))
	for _, answer := range poll.Answers {
		answers = append(answers, map[string]any{"id": answer.ID, "text": answer.Text, "votes": answer.Votes})
	}
	body := map[string]any{
		"owner_id": poll.OwnerID,
		"poll_id":  poll.PollID,
		"question": poll.Question,
		"answers":  answers,
	}
	raw, _ := json.Marshal(body)
	return string(raw)
}

func linkAttachJSON(url, host string) string {
	body := map[string]any{
		"url":   url,
		"title": host,
	}
	raw, _ := json.Marshal(body)
	return string(raw)
}

func noteAttachJSON(note model.Note) string {
	body := map[string]any{
		"owner_id": note.OwnerID,
		"nid":      note.NID,
		"title":    note.Title,
		"text":     note.Text,
	}
	raw, _ := json.Marshal(body)
	return string(raw)
}

func docAttachJSON(doc model.Doc) string {
	body := map[string]any{
		"owner_id": doc.OwnerID,
		"did":      doc.DID,
		"title":    doc.Title,
		"ext":      doc.Ext,
		"size":     doc.Size,
		"url":      doc.URL,
	}
	raw, _ := json.Marshal(body)
	return string(raw)
}
