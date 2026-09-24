package media

import (
	"net/http"
	"strconv"
	"strings"
)

func ServeBytes(w http.ResponseWriter, r *http.Request, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	start, end, ok := parseRange(r.Header.Get("Range"), len(data))
	if !ok {
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			w.Write(data)
		}
		return
	}

	chunk := data[start : end+1]
	w.Header().Set("Content-Length", strconv.Itoa(len(chunk)))
	w.Header().Set("Content-Range", "bytes "+strconv.Itoa(start)+"-"+strconv.Itoa(end)+"/"+strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusPartialContent)
	if r.Method != http.MethodHead {
		w.Write(chunk)
	}
}

func parseRange(header string, size int) (int, int, bool) {
	if header == "" || size <= 0 {
		return 0, 0, false
	}
	const prefix = "bytes="
	if !strings.HasPrefix(header, prefix) {
		return 0, 0, false
	}
	spec := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if strings.Contains(spec, ",") {
		spec = strings.SplitN(spec, ",", 2)[0]
	}
	dash := strings.Index(spec, "-")
	if dash < 0 {
		return 0, 0, false
	}
	from := strings.TrimSpace(spec[:dash])
	to := strings.TrimSpace(spec[dash+1:])

	switch {
	case from == "" && to == "":
		return 0, 0, false
	case from == "":
		length, err := strconv.Atoi(to)
		if err != nil || length <= 0 {
			return 0, 0, false
		}
		if length > size {
			length = size
		}
		return size - length, size - 1, true
	default:
		start, err := strconv.Atoi(from)
		if err != nil || start < 0 || start >= size {
			return 0, 0, false
		}
		end := size - 1
		if to != "" {
			parsed, err := strconv.Atoi(to)
			if err != nil || parsed < start {
				return 0, 0, false
			}
			if parsed < end {
				end = parsed
			}
		}
		return start, end, true
	}
}
