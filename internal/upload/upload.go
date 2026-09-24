package upload

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"vkemu/internal/model"
	"vkemu/internal/params"
	"vkemu/internal/store"
)

const (
	MaxSize     = 64 << 20
	DefaultName = "upload.bin"
)

type Part struct {
	Field    string
	Filename string
	Data     []byte
}

var fileFields = []string{"file", "file1", "photo", "video_file", "audio", "video", "document", "doc"}

func ReadParts(r *http.Request) ([]Part, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxSize))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, errors.New("empty body")
	}
	boundary := requestBoundary(r.Header.Get("Content-Type"))
	if boundary == "" {
		return []Part{{Filename: DefaultName, Data: body}}, nil
	}
	parts := splitMultipart(body, boundary)
	if len(parts) == 0 {
		return nil, errors.New("no parts")
	}
	return parts, nil
}

func ReadFile(r *http.Request) (Part, error) {
	parts, err := ReadParts(r)
	if err != nil {
		return Part{}, err
	}
	if part, ok := File(parts); ok {
		return part, nil
	}
	part := parts[0]
	if part.Filename == "" {
		part.Filename = DefaultName
	}
	return part, nil
}

func File(parts []Part) (Part, bool) {
	for _, field := range fileFields {
		for _, part := range parts {
			if part.Field == field && part.Filename != "" {
				return part, true
			}
		}
	}
	for _, part := range parts {
		if part.Filename != "" {
			return part, true
		}
	}
	return Part{}, false
}

func Field(parts []Part, name string) string {
	for _, part := range parts {
		if part.Field == name {
			return string(part.Data)
		}
	}
	return ""
}

func Save(db *store.Store, dir, kind string, ownerID int, part Part) (model.Upload, error) {
	if part.Filename == "" {
		part.Filename = DefaultName
	}
	hash := params.MD5(string(part.Data) + part.Filename)
	sub := filepath.Join(dir, hash[:2])
	if err := os.MkdirAll(sub, 0o755); err != nil {
		return model.Upload{}, err
	}
	path := filepath.Join(sub, hash+filepath.Ext(part.Filename))
	if err := os.WriteFile(path, part.Data, 0o644); err != nil {
		return model.Upload{}, err
	}
	item := model.Upload{
		Hash:    hash,
		Kind:    kind,
		OwnerID: ownerID,
		Path:    path,
		Size:    len(part.Data),
		Title:   part.Filename,
		Created: db.Now(),
	}
	if err := db.SaveUpload(item); err != nil {
		return model.Upload{}, err
	}
	return item, nil
}

func URL(item model.Upload) string {
	return "/media/upload/" + item.Hash + Extension(item.Title)
}

func Extension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ""
	}
	return ext
}

func ContentType(filename string) string {
	switch Extension(filename) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a", ".aac":
		return "audio/mp4"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".3gp":
		return "video/3gpp"
	case ".webm":
		return "video/webm"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain; charset=utf-8"
	}
	return "application/octet-stream"
}

func requestBoundary(contentType string) string {
	if contentType == "" {
		return ""
	}
	mediaType, mediaParams, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return ""
	}
	return mediaParams["boundary"]
}

func splitMultipart(body []byte, boundary string) []Part {
	delimiter := []byte("--" + boundary)
	chunks := bytes.Split(body, delimiter)
	out := make([]Part, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = trimBoundaryEdges(chunk)
		if len(chunk) == 0 {
			continue
		}
		header, data, ok := splitHeaders(chunk)
		if !ok {
			continue
		}
		part := parsePartHeaders(header)
		part.Data = data
		out = append(out, part)
	}
	return out
}

func trimBoundaryEdges(chunk []byte) []byte {
	chunk = bytes.TrimLeft(chunk, "\r\n")
	if bytes.HasPrefix(chunk, []byte("--")) {
		return nil
	}
	return bytes.TrimRight(chunk, "\r\n")
}

func splitHeaders(chunk []byte) ([]byte, []byte, bool) {
	index := bytes.Index(chunk, []byte("\r\n\r\n"))
	sep := 4
	if index < 0 {
		index = bytes.Index(chunk, []byte("\n\n"))
		sep = 2
	}
	if index < 0 {
		return nil, nil, false
	}
	return chunk[:index], chunk[index+sep:], true
}

func parsePartHeaders(raw []byte) Part {
	var part Part
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(strings.ToLower(line), "content-disposition:") {
			continue
		}
		value := strings.TrimSpace(line[len("content-disposition:"):])
		for _, field := range strings.Split(value, ";") {
			field = strings.TrimSpace(field)
			switch {
			case strings.HasPrefix(field, "name="):
				part.Field = unquote(strings.TrimPrefix(field, "name="))
			case strings.HasPrefix(field, "filename="):
				part.Filename = unquote(strings.TrimPrefix(field, "filename="))
			}
		}
	}
	return part
}

func unquote(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"`)
}
