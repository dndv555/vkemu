package api

import (
	"vkemu/internal/model"
)

func (s *Server) audioGet(c *call) (any, error) {
	uid := c.p.Int("uid", c.viewer())
	if uid == 0 {
		uid = c.viewer()
	}
	return s.audioList(c, uid, c.p.Int("album_id", 0))
}

func (s *Server) audioGetRecommendations(c *call) (any, error) {
	uid := c.p.Int("uid", c.viewer())
	if uid == 0 {
		uid = c.viewer()
	}
	items, err := s.db.SearchAudio("", 0, c.p.Int("count", 100))
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, c.formatAudio(item))
	}
	return out, nil
}

func (s *Server) audioList(c *call, uid, albumID int) (any, error) {
	offset := c.p.Int("offset", 0)
	count := c.p.Int("count", 100)
	if count <= 0 {
		count = 100
	}
	items, err := s.db.Audio(uid, offset, count, albumID)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, c.formatAudio(item))
	}
	return out, nil
}

func (s *Server) audioGetByID(c *call) (any, error) {
	items := c.p.List("audios")
	pairs := make([][2]int, 0, len(items))
	for _, item := range items {
		parts := splitPair(item)
		if parts == nil {
			continue
		}
		pairs = append(pairs, [2]int{parts[0], parts[1]})
	}
	audios, err := s.db.AudioByID(pairs)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(audios))
	for _, item := range audios {
		out = append(out, c.formatAudio(item))
	}
	return out, nil
}

func (s *Server) audioSearch(c *call) (any, error) {
	query := c.p.String("q", "")
	items, err := s.db.SearchAudio(query, c.p.Int("offset", 0), c.p.Int("count", 100))
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items)+1)
	out = append(out, len(items))
	for _, item := range items {
		out = append(out, c.formatAudio(item))
	}
	return out, nil
}

func (s *Server) audioGetAlbums(c *call) (any, error) {
	uid := c.p.Int("uid", c.viewer())
	if uid == 0 {
		uid = c.viewer()
	}
	albums, err := s.db.AudioAlbums(uid)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(albums)+1)
	out = append(out, len(albums))
	for _, album := range albums {
		out = append(out, map[string]any{
			"album_id": album.AlbumID,
			"owner_id": album.OwnerID,
			"title":    album.Title,
		})
	}
	return out, nil
}

func (s *Server) audioAdd(c *call) (any, error) {
	oid := c.p.Int("oid", 0)
	aid := c.p.Int("aid", 0)
	items, err := s.db.AudioByID([][2]int{{oid, aid}})
	if err != nil || len(items) == 0 {
		return nil, errNotFound
	}
	item := items[0]
	newAID, err := s.db.NextAudioID()
	if err != nil {
		return nil, err
	}
	item.OwnerID = c.viewer()
	item.AID = newAID
	item.AlbumID = 0
	if err := s.db.AddAudio(item); err != nil {
		return nil, err
	}
	return newAID, nil
}

func (s *Server) audioDelete(c *call) (any, error) {
	oid := c.p.Int("oid", c.viewer())
	aid := c.p.Int("aid", 0)
	if err := s.db.DeleteAudio(oid, aid); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) videoGet(c *call) (any, error) {
	items := c.p.List("videos")
	if len(items) == 0 {
		ownerID := c.p.Int("owner_id", c.viewer())
		videos, err := s.db.Videos(ownerID, c.p.Int("offset", 0), c.p.Int("count", 100))
		if err != nil {
			return nil, err
		}
		out := make([]any, 0, len(videos)+1)
		out = append(out, len(videos))
		for _, video := range videos {
			out = append(out, c.formatVideo(video))
		}
		return out, nil
	}
	out := make([]any, 0, len(items)+1)
	out = append(out, len(items))
	for _, item := range items {
		parts := splitPair(item)
		if parts == nil {
			continue
		}
		video, err := s.db.Video(parts[0], parts[1])
		if err != nil {
			continue
		}
		out = append(out, c.formatVideo(video))
	}
	return out, nil
}

func (s *Server) videoSave(c *call) (any, error) {
	vid, err := s.db.NextVideoID()
	if err != nil {
		return nil, err
	}
	video := model.Video{
		OwnerID:     c.viewer(),
		VID:         vid,
		Title:       c.p.String("name", "Видео"),
		Description: c.p.String("description", ""),
		Date:        s.db.Now(),
	}
	if err := s.db.AddVideo(video); err != nil {
		return nil, err
	}
	result := c.formatVideo(video)
	result["upload_url"] = c.abs("/upload/video?oid=" + itoa(video.OwnerID) + "&vid=" + itoa(video.VID))
	return result, nil
}

func (s *Server) notesGetByID(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	nid := c.p.Int("nid", 0)
	note, err := s.db.Note(ownerID, nid)
	if err != nil {
		return nil, errNotFound
	}
	return c.formatNote(note), nil
}

func (s *Server) groupsGetByID(c *call) (any, error) {
	ids := c.p.IntList("gids")
	if len(ids) == 0 {
		ids = c.p.IntList("group_ids")
	}
	groups, err := s.db.Groups(ids)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(groups))
	for _, group := range groups {
		out = append(out, c.formatGroup(group))
	}
	return out, nil
}

func (s *Server) groupsGetFull(c *call) (any, error) {
	return s.groupsGetByID(c)
}

func (s *Server) statusGet(c *call) (any, error) {
	uid := c.p.Int("uid", c.viewer())
	if uid == 0 {
		uid = c.viewer()
	}
	return c.formatStatus(uid), nil
}

func (s *Server) statusSet(c *call) (any, error) {
	if err := s.db.UpdateUserStatus(c.viewer(), c.p.String("text", "")); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) pollsGetByID(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	pollID := c.p.Int("poll_id", 0)
	poll, err := s.db.Poll(ownerID, pollID)
	if err != nil {
		return nil, errNotFound
	}
	return c.formatPoll(poll, c.viewer()), nil
}

func (s *Server) pollsAddVote(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	pollID := c.p.Int("poll_id", 0)
	answerID := c.p.Int("answer_id", 0)
	if err := s.db.PollVote(ownerID, pollID, c.viewer(), answerID); err != nil {
		return nil, err
	}
	return 1, nil
}
