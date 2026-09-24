package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"vkemu/internal/model"
	"vkemu/internal/upload"
)

func (s *Server) photosGetUploadServer(c *call) (any, error) {
	return map[string]any{
		"upload_url": c.abs("/upload/photo?aid=" + c.p.String("aid", "0")),
		"aid":        c.p.Int("aid", 0),
		"user_id":    c.viewer(),
	}, nil
}

func (s *Server) photosGetWallUploadServer(c *call) (any, error) {
	return map[string]any{
		"upload_url": c.abs("/upload/wall_photo"),
		"user_id":    c.viewer(),
	}, nil
}

func (s *Server) audioGetUploadServer(c *call) (any, error) {
	return map[string]any{"upload_url": c.abs("/upload/audio")}, nil
}

func (s *Server) docsGetUploadServer(c *call) (any, error) {
	return map[string]any{"upload_url": c.abs("/upload/doc")}, nil
}

func (s *Server) photosSave(c *call) (any, error) {
	aid := c.p.Int("aid", 0)
	item, ok := s.db.Upload(c.p.String("photo", c.p.String("file", "")))
	if !ok {
		return nil, errNotFound
	}
	pid, err := s.db.NextPhotoID()
	if err != nil {
		return nil, err
	}
	photo := model.Photo{
		OwnerID: c.viewer(),
		PID:     pid,
		AID:     aid,
		Date:    s.db.Now(),
		Text:    c.p.String("caption", ""),
		Src:     upload.URL(item),
		SrcBig:  upload.URL(item),
	}
	if err := s.db.AddPhoto(photo); err != nil {
		return nil, err
	}
	return []any{c.formatPhoto(photo)}, nil
}

func (s *Server) photosSaveWallPhoto(c *call) (any, error) {
	item, ok := s.db.Upload(c.p.String("photo", c.p.String("file", "")))
	if !ok {
		return nil, errNotFound
	}
	ownerID := c.p.Int("uid", c.viewer())
	if ownerID == 0 {
		ownerID = c.viewer()
	}
	pid, err := s.db.NextPhotoID()
	if err != nil {
		return nil, err
	}
	photo := model.Photo{
		OwnerID: ownerID,
		PID:     pid,
		AID:     -6,
		Date:    s.db.Now(),
		Src:     upload.URL(item),
		SrcBig:  upload.URL(item),
	}
	if err := s.db.AddPhoto(photo); err != nil {
		return nil, err
	}
	return []any{c.formatPhoto(photo)}, nil
}

func (s *Server) audioSave(c *call) (any, error) {
	item, ok := s.db.Upload(c.p.String("audio", c.p.String("file", "")))
	if !ok {
		return nil, errNotFound
	}
	aid, err := s.db.NextAudioID()
	if err != nil {
		return nil, err
	}
	title := item.Title
	if title == "" {
		title = "Загруженный трек"
	}
	audio := model.Audio{
		OwnerID:  c.viewer(),
		AID:      aid,
		Artist:   c.p.String("artist", "Unknown"),
		Title:    c.p.String("title", strings.TrimSuffix(title, filepath.Ext(title))),
		Duration: 30,
		URL:      upload.URL(item),
	}
	if err := s.db.AddAudio(audio); err != nil {
		return nil, err
	}
	return c.formatAudio(audio), nil
}

func (s *Server) docsSave(c *call) (any, error) {
	item, ok := s.db.Upload(c.p.String("file", ""))
	if !ok {
		return nil, errNotFound
	}
	did, err := s.db.NextDocID()
	if err != nil {
		return nil, err
	}
	title := item.Title
	if title == "" {
		title = "document"
	}
	doc := model.Doc{
		OwnerID: c.viewer(),
		DID:     did,
		Title:   title,
		Ext:     strings.TrimPrefix(filepath.Ext(title), "."),
		Size:    item.Size,
		URL:     upload.URL(item),
		Date:    s.db.Now(),
	}
	if err := s.db.AddDoc(doc); err != nil {
		return nil, err
	}
	return []any{c.formatDoc(doc)}, nil
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimPrefix(r.URL.Path, "/upload/")
	if kind == "" {
		kind = "file"
	}
	ownerID := atoiDefault(r.URL.Query().Get("uid"), 0)

	parts, err := upload.ReadParts(r)
	if err != nil {
		writeError(w, errf(100, "Bad upload: %s", err.Error()))
		return
	}
	part, ok := upload.File(parts)
	if !ok {
		writeError(w, errf(100, "Bad upload: no file part"))
		return
	}

	item, err := upload.Save(s.db, s.cfg.UploadDir, kind, ownerID, part)
	if err != nil {
		writeError(w, errf(1, "Store upload: %s", err.Error()))
		return
	}

	switch {
	case kind == "avatar":
		writeJSON(w, map[string]any{"server": 1, "photo": item.Hash, "hash": item.Hash, "url": upload.URL(item)})
	case strings.Contains(kind, "audio"):
		writeJSON(w, map[string]any{"server": 1, "audio": item.Hash, "hash": item.Hash})
	case strings.Contains(kind, "doc"):
		writeJSON(w, map[string]any{"file": item.Hash})
	case strings.Contains(kind, "video"):
		s.linkVideo(r, item)
		writeJSON(w, map[string]any{"server": 1, "video": item.Hash, "hash": item.Hash})
	default:
		writeJSON(w, map[string]any{
			"server":      1,
			"photo":       item.Hash,
			"hash":        item.Hash,
			"photos_list": "[]",
			"aid":         atoiDefault(r.URL.Query().Get("aid"), 0),
		})
	}
}

func (s *Server) linkVideo(r *http.Request, item model.Upload) {
	query := r.URL.Query()
	ownerID := atoiDefault(query.Get("oid"), 0)
	vid := atoiDefault(query.Get("vid"), 0)
	if ownerID == 0 || vid == 0 {
		return
	}
	video, err := s.db.Video(ownerID, vid)
	if err != nil {
		return
	}
	video.URL = upload.URL(item)
	if err := s.db.UpdateVideo(video); err != nil {
		return
	}
}

func (s *Server) handleCaptcha(w http.ResponseWriter, r *http.Request) {
	png, err := captchaImage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func (s *Server) captchaForce(c *call) (any, error) {
	return 1, nil
}

func (s *Server) handleLongPoll(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	key := query.Get("key")
	uid, ok := s.hub.UIDByKey(key)
	if !ok {
		writeJSON(w, map[string]any{"failed": 1})
		return
	}
	ts := atoiDefault(query.Get("ts"), 0)
	if ts <= 0 {
		ts = s.hub.TS(uid)
	}

	updates := []any{}
	if ts > 0 {
		newTS, pending := s.hub.Wait(uid, ts, waitDuration(s.cfg.LongPollWait))
		ts = newTS
		updates = pending
	}
	if updates == nil {
		updates = []any{}
	}
	writeJSON(w, map[string]any{"ts": ts, "updates": updates})
}

func atoiDefault(value string, def int) int {
	if value == "" {
		return def
	}
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	return n
}
