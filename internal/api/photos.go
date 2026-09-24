package api

import (
	"vkemu/internal/model"
)

func (s *Server) photosGet(c *call) (any, error) {
	uid := c.p.Int("uid", c.viewer())
	if uid == 0 {
		uid = c.viewer()
	}
	aid := c.p.Int("aid", 0)
	return s.photosList(c, uid, aid)
}

func (s *Server) photosGetUserPhotos(c *call) (any, error) {
	uid := c.p.Int("uid", c.viewer())
	if uid == 0 {
		uid = c.viewer()
	}
	photos, err := s.db.Photos(uid, 0, c.p.Int("offset", 0), c.p.Int("count", 100))
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(photos)+1)
	out = append(out, len(photos))
	for _, photo := range photos {
		out = append(out, c.formatPhoto(photo))
	}
	return out, nil
}

func (s *Server) photosList(c *call, uid, aid int) (any, error) {
	photos, err := s.db.Photos(uid, aid, c.p.Int("offset", 0), c.p.Int("count", 100))
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(photos))
	for _, photo := range photos {
		out = append(out, c.formatPhoto(photo))
	}
	return out, nil
}

func (s *Server) photosGetByID(c *call) (any, error) {
	items := c.p.List("photos")
	pairs := make([][2]int, 0, len(items))
	for _, item := range items {
		parts := splitPair(item)
		if parts == nil {
			continue
		}
		pairs = append(pairs, [2]int{parts[0], parts[1]})
	}
	photos, err := s.db.PhotoByID(pairs)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(photos))
	for _, photo := range photos {
		out = append(out, c.formatPhoto(photo))
	}
	return out, nil
}

func (s *Server) photosGetAlbums(c *call) (any, error) {
	uid := c.p.Int("uid", c.viewer())
	if uid == 0 {
		uid = c.viewer()
	}
	albums, err := s.db.Albums(uid)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(albums))
	for _, album := range albums {
		out = append(out, c.formatAlbum(album))
	}
	return out, nil
}

func (s *Server) photosCreateAlbum(c *call) (any, error) {
	aid, err := s.db.NextAlbumID()
	if err != nil {
		return nil, err
	}
	album := model.Album{
		OwnerID:     c.viewer(),
		AID:         aid,
		Title:       c.p.String("title", "Новый альбом"),
		Description: c.p.String("description", ""),
		Privacy:     c.p.Int("privacy", 0),
		ThumbID:     -1,
		Created:     s.db.Now(),
	}
	if err := s.db.AddAlbum(album); err != nil {
		return nil, err
	}
	return c.formatAlbum(album), nil
}

func (s *Server) photosEditAlbum(c *call) (any, error) {
	aid := c.p.Int("aid", 0)
	album, err := s.db.Album(c.viewer(), aid)
	if err != nil {
		return nil, errNotFound
	}
	title := c.p.String("title", album.Title)
	description := c.p.String("description", album.Description)
	privacy := c.p.Int("privacy", album.Privacy)
	if err := s.db.UpdateAlbum(c.viewer(), aid, title, description, privacy); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) photosDeleteAlbum(c *call) (any, error) {
	aid := c.p.Int("aid", 0)
	if err := s.db.DeleteAlbum(c.viewer(), aid); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) photosDelete(c *call) (any, error) {
	ownerID := c.p.Int("oid", c.viewer())
	pid := c.p.Int("pid", 0)
	if err := s.db.DeletePhoto(ownerID, pid); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) photosEdit(c *call) (any, error) {
	ownerID := c.p.Int("uid", c.viewer())
	pid := c.p.Int("pid", 0)
	if err := s.db.SetPhotoText(ownerID, pid, c.p.String("caption", "")); err != nil {
		return nil, err
	}
	return 1, nil
}

func (s *Server) photosCreateComment(c *call) (any, error) {
	ownerID := c.p.Int("owner_id", c.viewer())
	pid := c.p.Int("pid", 0)
	comment := model.Comment{
		CType:   "photo",
		OwnerID: ownerID,
		ItemID:  pid,
		FromID:  c.viewer(),
		Date:    s.db.Now(),
		Text:    c.p.String("message", ""),
	}
	if err := s.db.AddComment(comment); err != nil {
		return nil, err
	}
	list, err := s.db.Comments("photo", ownerID, pid, 0, 1000)
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		return list[len(list)-1].CID, nil
	}
	return 0, nil
}

func (s *Server) photosGetTags(c *call) (any, error) {
	return []any{}, nil
}

func splitPair(value string) []int {
	parts := splitBy(value, '_')
	if len(parts) != 2 {
		return nil
	}
	a, ok := parseUID(parts[0])
	if !ok {
		return nil
	}
	b, ok := parseUID(parts[1])
	if !ok {
		return nil
	}
	return []int{a, b}
}

func splitBy(value string, sep rune) []string {
	var out []string
	current := ""
	for _, r := range value {
		if r == sep {
			out = append(out, current)
			current = ""
			continue
		}
		current += string(r)
	}
	out = append(out, current)
	return out
}
