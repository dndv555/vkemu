package web

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"vkemu/internal/model"
	"vkemu/internal/upload"
)

func (s *Server) saveUpload(kind string, ownerID int, part upload.Part) (model.Upload, error) {
	return upload.Save(s.db, s.uploadDir, kind, ownerID, part)
}

func (s *Server) savePhoto(ownerID int, part upload.Part) (model.Photo, error) {
	item, err := s.saveUpload("wall_photo", ownerID, part)
	if err != nil {
		return model.Photo{}, err
	}
	pid, err := s.db.NextPhotoID()
	if err != nil {
		return model.Photo{}, err
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
		return model.Photo{}, err
	}
	return photo, nil
}

func (s *Server) saveAudio(ownerID int, artist, title string, part upload.Part) (model.Audio, error) {
	item, err := s.saveUpload("audio", ownerID, part)
	if err != nil {
		return model.Audio{}, err
	}
	aid, err := s.db.NextAudioID()
	if err != nil {
		return model.Audio{}, err
	}
	if artist == "" {
		artist = "Unknown"
	}
	if title == "" {
		title = strings.TrimSuffix(item.Title, filepath.Ext(item.Title))
	}
	audio := model.Audio{
		OwnerID: ownerID,
		AID:     aid,
		Artist:  artist,
		Title:   title,
		URL:     upload.URL(item),
	}
	if err := s.db.AddAudio(audio); err != nil {
		return model.Audio{}, err
	}
	return audio, nil
}

func (s *Server) saveVideo(ownerID int, title string, part upload.Part) (model.Video, error) {
	item, err := s.saveUpload("video", ownerID, part)
	if err != nil {
		return model.Video{}, err
	}
	vid, err := s.db.NextVideoID()
	if err != nil {
		return model.Video{}, err
	}
	if title == "" {
		title = "Видео"
	}
	video := model.Video{
		OwnerID: ownerID,
		VID:     vid,
		Title:   title,
		URL:     upload.URL(item),
		Date:    s.db.Now(),
	}
	if err := s.db.AddVideo(video); err != nil {
		return model.Video{}, err
	}
	return video, nil
}

func (s *Server) saveDoc(ownerID int, part upload.Part) (model.Doc, error) {
	item, err := s.saveUpload("doc", ownerID, part)
	if err != nil {
		return model.Doc{}, err
	}
	did, err := s.db.NextDocID()
	if err != nil {
		return model.Doc{}, err
	}
	doc := model.Doc{
		OwnerID: ownerID,
		DID:     did,
		Title:   item.Title,
		Ext:     strings.TrimPrefix(filepath.Ext(item.Title), "."),
		Size:    item.Size,
		URL:     upload.URL(item),
		Date:    s.db.Now(),
	}
	if err := s.db.AddDoc(doc); err != nil {
		return model.Doc{}, err
	}
	return doc, nil
}

func (s *Server) uploadAttachment(ownerID int, part upload.Part) (string, string, error) {
	kind := strings.ToLower(strings.TrimPrefix(filepath.Ext(part.Filename), "."))
	switch kind {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp":
		photo, err := s.savePhoto(ownerID, part)
		if err != nil {
			return "", "", err
		}
		return "photo", photoAttachJSON(photo), nil
	case "mp3", "m4a", "aac", "ogg", "oga", "wav":
		audio, err := s.saveAudio(ownerID, "Unknown", "", part)
		if err != nil {
			return "", "", err
		}
		return "audio", audioAttachJSON(audio), nil
	case "mp4", "m4v", "3gp", "webm":
		video, err := s.saveVideo(ownerID, "", part)
		if err != nil {
			return "", "", err
		}
		return "video", videoAttachJSON(video), nil
	default:
		doc, err := s.saveDoc(ownerID, part)
		if err != nil {
			return "", "", err
		}
		return "doc", docAttachJSON(doc), nil
	}
}

func photoAttachJSON(photo model.Photo) string {
	raw, _ := json.Marshal(map[string]any{
		"owner_id": photo.OwnerID,
		"pid":      photo.PID,
		"aid":      photo.AID,
		"src":      photo.Src,
		"src_big":  photo.SrcBig,
		"text":     photo.Text,
	})
	return string(raw)
}

func audioAttachJSON(item model.Audio) string {
	raw, _ := json.Marshal(map[string]any{
		"owner_id": item.OwnerID,
		"aid":      item.AID,
		"artist":   item.Artist,
		"title":    item.Title,
		"duration": item.Duration,
		"url":      item.URL,
	})
	return string(raw)
}

func videoAttachJSON(video model.Video) string {
	raw, _ := json.Marshal(map[string]any{
		"owner_id": video.OwnerID,
		"vid":      video.VID,
		"title":    video.Title,
		"image":    video.Image,
		"url":      video.URL,
	})
	return string(raw)
}

func docAttachJSON(doc model.Doc) string {
	raw, _ := json.Marshal(map[string]any{
		"owner_id": doc.OwnerID,
		"did":      doc.DID,
		"title":    doc.Title,
		"ext":      doc.Ext,
		"size":     doc.Size,
		"url":      doc.URL,
	})
	return string(raw)
}
