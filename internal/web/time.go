package web

import "time"

func formatTime(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).Format("02.01.2006 15:04")
}
