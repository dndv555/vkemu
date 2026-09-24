package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"vkemu/internal/media"
	"vkemu/internal/upload"
)

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/media/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	kind := parts[0]
	owner := parts[1]
	name := parts[len(parts)-1]

	switch kind {
	case "upload":
		s.serveUpload(w, r, name)
	case "audio":
		media.ServeBytes(w, r, "audio/wav", media.SilenceWAV(30))
	case "photo", "video", "doc":
		size := 600
		if strings.HasSuffix(name, "_big.png") {
			size = 800
		}
		if kind == "video" {
			size = 480
		}
		png, err := media.Placeholder(kind+":"+owner+":"+name, size)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		media.ServeBytes(w, r, "image/png", png)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) serveUpload(w http.ResponseWriter, r *http.Request, name string) {
	hash := strings.TrimSuffix(name, filepath.Ext(name))
	item, ok := s.db.Upload(hash)
	if !ok {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(item.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	media.ServeBytes(w, r, upload.ContentType(item.Title), data)
}
