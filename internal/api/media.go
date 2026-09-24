package api

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"vkemu/internal/media"
	uploadpkg "vkemu/internal/upload"
)

func waitDuration(seconds int) time.Duration {
	if seconds <= 0 {
		seconds = 25
	}
	return time.Duration(seconds) * time.Second
}

func captchaImage() ([]byte, error) {
	return media.Placeholder("captcha", 120)
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/media/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	kind := parts[0]
	ownerID := parts[1]
	name := parts[len(parts)-1]

	switch kind {
	case "audio":
		media.ServeBytes(w, r, "audio/wav", media.SilenceWAV(30))
	case "photo", "video", "doc":
		seed := kind + ":" + ownerID + ":" + name
		size := 600
		if strings.HasSuffix(name, "_big.png") {
			size = 800
		}
		if kind == "video" {
			size = 480
		}
		png, err := media.Placeholder(seed, size)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		media.ServeBytes(w, r, "image/png", png)
	case "upload":
		s.serveUpload(w, r, ownerID)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) serveUpload(w http.ResponseWriter, r *http.Request, hash string) {
	upload, ok := s.db.Upload(strings.TrimSuffix(hash, filepath.Ext(hash)))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", uploadpkg.ContentType(upload.Title))
	http.ServeFile(w, r, upload.Path)
}
