package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"vkemu/internal/model"
)

func (c *call) userPhoto(uid int, size string) string {
	if user, err := c.srv.db.User(uid); err == nil && user.Avatar != "" {
		return c.abs(user.Avatar)
	}
	suffix := ""
	if size == "big" {
		suffix = "_big"
	}
	return c.abs("/media/photo/" + strconv.Itoa(uid) + "/" + strconv.Itoa(uid) + suffix + ".png")
}

func (c *call) mediaURL(path string, fallback func() string) string {
	if path == "" {
		return c.abs(fallback())
	}
	return c.abs(path)
}

func (c *call) photoURL(ownerID, pid int, big bool) string {
	suffix := ""
	if big {
		suffix = "_big"
	}
	return c.abs("/media/photo/" + strconv.Itoa(ownerID) + "/" + strconv.Itoa(pid) + suffix + ".png")
}

func (c *call) audioURL(ownerID, aid int) string {
	return c.abs("/media/audio/" + strconv.Itoa(ownerID) + "/" + strconv.Itoa(aid) + ".wav")
}

func (c *call) docURL(ownerID, did int, ext string) string {
	if ext == "" {
		ext = "txt"
	}
	return c.abs("/media/doc/" + strconv.Itoa(ownerID) + "/" + strconv.Itoa(did) + "." + ext)
}

func (c *call) formatUser(u model.User, nameCase string) map[string]any {
	first := declineFirstName(u.FirstName, u.Sex, nameCase)
	last := declineLastName(u.LastName, u.Sex, nameCase)
	photo := c.userPhoto(u.UID, "rec")

	obj := map[string]any{
		"uid":              u.UID,
		"first_name":       first,
		"last_name":        last,
		"nickname":         u.Nickname,
		"domain":           u.Domain,
		"sex":              u.Sex,
		"bdate":            u.BDate,
		"city":             u.City,
		"country":          u.Country,
		"photo":            photo,
		"photo_rec":        photo,
		"photo_medium":     photo,
		"photo_medium_rec": photo,
		"photo_big":        c.userPhoto(u.UID, "big"),
		"online":           boolInt(u.Online),
		"online_mobile":    0,
		"last_seen": map[string]any{
			"time":     int(time.Now().Unix() - 300),
			"platform": 7,
		},
		"can_post":                  1,
		"can_see_all_posts":         1,
		"can_write_private_message": 1,
		"mobile_phone":              u.MobilePhone,
		"home_phone":                u.HomePhone,
		"university_name":           u.UniversityName,
		"graduation":                u.Graduation,
		"relation":                  u.Relation,
		"status":                    u.Status,
		"counters": map[string]any{
			"friends": 0,
			"photos":  0,
			"albums":  0,
			"audios":  0,
			"videos":  0,
			"notes":   0,
		},
	}
	obj["counters"] = c.userCounters(u.UID)
	return obj
}

func (c *call) userCounters(uid int) map[string]any {
	friends := 0
	if ids, err := c.srv.db.FriendIDs(uid); err == nil {
		friends = len(ids)
	}
	albums := 0
	if list, err := c.srv.db.Albums(uid); err == nil {
		albums = len(list)
	}
	audios := 0
	if list, err := c.srv.db.Audio(uid, 0, 1000, 0); err == nil {
		audios = len(list)
	}
	mutual := 0
	if ids, err := c.srv.db.MutualFriendIDs(c.viewer(), uid); err == nil {
		mutual = len(ids)
	}
	return map[string]any{
		"friends":        friends,
		"mutual_friends": mutual,
		"albums":         albums,
		"photos":         c.srv.db.PhotoCountAll(uid),
		"user_photos":    c.srv.db.PhotoCountAll(uid),
		"audios":         audios,
		"videos":         1,
		"notes":          1,
	}
}

func (c *call) formatStatus(uid int) map[string]any {
	status := ""
	if user, err := c.srv.db.User(uid); err == nil {
		status = user.Status
	}
	return map[string]any{"text": status}
}

func (c *call) formatPost(post model.WallPost, viewer int) map[string]any {
	obj := map[string]any{
		"id":       post.PostID,
		"post_id":  post.PostID,
		"owner_id": post.OwnerID,
		"from_id":  post.FromID,
		"date":     int(post.Date),
		"text":     post.Text,
		"comments": map[string]any{
			"count":    post.Comments,
			"can_post": 1,
		},
		"likes": map[string]any{
			"count":      post.Likes,
			"user_likes": boolInt(c.srv.db.IsLiked("post", post.OwnerID, post.PostID, viewer)),
		},
		"can_delete":  boolInt(post.FromID == viewer || post.OwnerID == viewer),
		"can_edit":    boolInt(post.FromID == viewer),
		"post_source": map[string]any{"type": "vk"},
		"online":      0,
	}
	if post.CopyOwnerID != 0 {
		obj["copy_owner_id"] = post.CopyOwnerID
		obj["copy_post_id"] = post.CopyPostID
	}
	if post.AttachType != "" && post.AttachJSON != "" {
		var attachment map[string]any
		if err := json.Unmarshal([]byte(post.AttachJSON), &attachment); err == nil {
			c.absolutizeAttachment(post.AttachType, attachment)
			obj["attachment"] = map[string]any{
				"type":          post.AttachType,
				post.AttachType: attachment,
			}
		}
	}
	if post.GeoJSON != "" {
		var geo map[string]any
		if err := json.Unmarshal([]byte(post.GeoJSON), &geo); err == nil {
			obj["geo"] = geo
		}
	}
	return obj
}

func (c *call) absolutizeAttachment(kind string, data map[string]any) {
	for _, key := range []string{"src", "src_big", "src_small", "image", "url", "photo"} {
		value, ok := data[key].(string)
		if !ok || value == "" {
			continue
		}
		data[key] = c.abs(value)
	}
	switch kind {
	case "audio":
		if _, ok := data["performer"]; !ok {
			data["performer"] = data["artist"]
		}
		if _, ok := data["artist"]; !ok {
			data["artist"] = data["performer"]
		}
	case "photo":
		if _, ok := data["src"]; !ok {
			data["src"] = data["src_big"]
		}
	}
}

func (c *call) formatComment(comment model.Comment) map[string]any {
	return map[string]any{
		"cid":          comment.CID,
		"id":           comment.CID,
		"from_id":      comment.FromID,
		"uid":          comment.FromID,
		"date":         int(comment.Date),
		"text":         comment.Text,
		"message":      comment.Text,
		"reply_to_uid": comment.ReplyToUID,
		"reply_to_cid": 0,
	}
}

func (c *call) formatMessage(msg model.Message, viewer int) map[string]any {
	peer := msg.PeerFor(viewer)
	out := msg.OutFor(viewer)
	obj := map[string]any{
		"mid":        msg.MID,
		"uid":        peer,
		"from_id":    msg.FromID,
		"date":       int(msg.Date),
		"body":       msg.Body,
		"read_state": boolInt(msg.ReadState),
		"out":        boolInt(out),
		"title":      "",
	}
	if msg.ChatID != 0 {
		obj["chat_id"] = msg.ChatID
		obj["title"] = c.srv.db.ChatTitle(msg.ChatID)
		members, _ := c.srv.db.ChatMembers(msg.ChatID)
		parts := make([]string, 0, len(members))
		for _, uid := range members {
			if uid != viewer {
				parts = append(parts, strconv.Itoa(uid))
			}
		}
		obj["chat_active"] = strings.Join(parts, ",")
		obj["uid"] = 2000000000 + msg.ChatID
	}
	if msg.AttachJSON != "" {
		var attachments []map[string]any
		if err := json.Unmarshal([]byte(msg.AttachJSON), &attachments); err == nil {
			for _, attachment := range attachments {
				kind, _ := attachment["type"].(string)
				if data, ok := attachment[kind].(map[string]any); ok {
					c.absolutizeAttachment(kind, data)
				}
			}
			obj["attachments"] = attachments
		}
	}
	return obj
}

func (c *call) formatAudio(item model.Audio) map[string]any {
	url := c.mediaURL(item.URL, func() string { return c.audioURL(item.OwnerID, item.AID) })
	return map[string]any{
		"aid":       item.AID,
		"owner_id":  item.OwnerID,
		"artist":    item.Artist,
		"performer": item.Artist,
		"title":     item.Title,
		"duration":  item.Duration,
		"url":       url,
		"lyrics_id": 0,
		"album":     item.AlbumID,
		"genre":     0,
	}
}

func (c *call) formatAlbum(album model.Album) map[string]any {
	size := c.srv.db.PhotoCount(album.OwnerID, album.AID)
	thumb := album.ThumbID
	if thumb <= 0 {
		thumb = -1
	}
	return map[string]any{
		"aid":         album.AID,
		"owner_id":    album.OwnerID,
		"title":       album.Title,
		"description": album.Description,
		"privacy":     album.Privacy,
		"size":        size,
		"thumb_id":    thumb,
		"created":     int(album.Created),
		"updated":     int(album.Created),
	}
}

func (c *call) formatPhoto(photo model.Photo) map[string]any {
	src := c.mediaURL(photo.Src, func() string { return c.photoURL(photo.OwnerID, photo.PID, false) })
	big := c.mediaURL(photo.SrcBig, func() string { return c.photoURL(photo.OwnerID, photo.PID, true) })
	return map[string]any{
		"pid":       photo.PID,
		"aid":       photo.AID,
		"owner_id":  photo.OwnerID,
		"src":       src,
		"src_big":   big,
		"src_small": src,
		"created":   int(photo.Date),
		"text":      photo.Text,
		"width":     600,
		"height":    600,
		"likes": map[string]any{
			"count":      photo.Likes,
			"user_likes": boolInt(c.srv.db.IsLiked("photo", photo.OwnerID, photo.PID, c.viewer())),
		},
		"comments": map[string]any{
			"count":    c.srv.db.CommentCount("photo", photo.OwnerID, photo.PID),
			"can_post": 1,
		},
	}
}

func (c *call) formatVideo(video model.Video) map[string]any {
	image := c.mediaURL(video.Image, func() string { return c.photoURL(video.OwnerID, video.VID, false) })
	url := video.URL
	if url == "" {
		url = "/media/video/" + strconv.Itoa(video.OwnerID) + "/" + strconv.Itoa(video.VID) + ".mp4"
	}
	url = c.abs(url)
	return map[string]any{
		"vid":         video.VID,
		"owner_id":    video.OwnerID,
		"title":       video.Title,
		"description": video.Description,
		"duration":    video.Duration,
		"image":       image,
		"date":        int(video.Date),
		"views":       1,
		"player":      url,
		"files": map[string]any{
			"mp4_240": url,
			"mp4_360": url,
		},
	}
}

func (c *call) formatNote(note model.Note) map[string]any {
	return map[string]any{
		"nid":      note.NID,
		"owner_id": note.OwnerID,
		"title":    note.Title,
		"text":     note.Text,
		"date":     int(note.Date),
		"comments": c.srv.db.CommentCount("note", note.OwnerID, note.NID),
	}
}

func (c *call) formatDoc(doc model.Doc) map[string]any {
	url := c.mediaURL(doc.URL, func() string { return c.docURL(doc.OwnerID, doc.DID, doc.Ext) })
	return map[string]any{
		"did":      doc.DID,
		"owner_id": doc.OwnerID,
		"title":    doc.Title,
		"ext":      doc.Ext,
		"size":     doc.Size,
		"url":      url,
		"date":     int(doc.Date),
	}
}

func (c *call) formatGroup(group model.Group) map[string]any {
	photo := c.abs("/media/photo/" + strconv.Itoa(-group.GID) + "/" + strconv.Itoa(group.GID) + ".png")
	return map[string]any{
		"gid":          group.GID,
		"name":         group.Name,
		"photo":        photo,
		"photo_medium": photo,
		"is_closed":    boolInt(group.IsClosed),
		"is_admin":     boolInt(group.IsAdmin),
		"type":         "group",
	}
}

func (c *call) formatPlace(place model.Place) map[string]any {
	return map[string]any{
		"place_id":   place.PlaceID,
		"title":      place.Title,
		"address":    place.Address,
		"latitude":   place.Latitude,
		"longitude":  place.Longitude,
		"type":       place.Type,
		"checkins":   place.Checkins,
		"city_id":    place.CityID,
		"country_id": place.CountryID,
	}
}

func (c *call) formatPoll(poll model.Poll, viewer int) map[string]any {
	total := c.srv.db.PollTotalVotes(poll.OwnerID, poll.PollID)
	answers := make([]any, 0, len(poll.Answers))
	for _, answer := range poll.Answers {
		rate := 0.0
		if total > 0 {
			rate = float64(answer.Votes) / float64(total) * 100
		}
		answers = append(answers, map[string]any{
			"id":    answer.ID,
			"text":  answer.Text,
			"votes": answer.Votes,
			"rate":  round2(rate),
		})
	}
	return map[string]any{
		"poll_id":   poll.PollID,
		"owner_id":  poll.OwnerID,
		"question":  poll.Question,
		"created":   int(time.Now().Unix() - 3600),
		"votes":     total,
		"answer_id": c.srv.db.PollAnswerID(poll.OwnerID, poll.PollID, viewer),
		"answers":   answers,
	}
}

func (c *call) formatCity(city model.City) map[string]any {
	return map[string]any{"cid": city.CID, "name": city.Name, "important": 0}
}

func (c *call) formatCountry(country model.Country) map[string]any {
	return map[string]any{"cid": country.CID, "name": country.Name}
}

func (c *call) profileNames(uids []int) map[int]model.User {
	out := make(map[int]model.User, len(uids))
	users, err := c.srv.db.Users(uids)
	if err != nil {
		return out
	}
	for _, user := range users {
		out[user.UID] = user
	}
	return out
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func declineFirstName(name string, sex int, nameCase string) string {
	return decline(name, sex, nameCase)
}

func declineLastName(name string, sex int, nameCase string) string {
	return decline(name, sex, nameCase)
}

func decline(name string, sex int, nameCase string) string {
	if name == "" || nameCase == "" || nameCase == "nom" {
		return name
	}
	lower := strings.ToLower(name)
	female := sex == 1 || strings.HasSuffix(lower, "а") || strings.HasSuffix(lower, "я")
	switch nameCase {
	case "gen":
		if female {
			return replaceTail(name, "ы", "и")
		}
		return name + "а"
	case "dat":
		if female {
			return replaceTail(name, "е", "е")
		}
		return name + "у"
	case "acc":
		if female {
			return replaceTail(name, "у", "ю")
		}
		return name + "а"
	case "ins":
		if female {
			return replaceTail(name, "ой", "ей")
		}
		return name + "ом"
	case "abl":
		if female {
			return replaceTail(name, "ой", "ей")
		}
		return name + "е"
	}
	return name
}

func replaceTail(name, soft, hard string) string {
	switch {
	case strings.HasSuffix(name, "а"):
		return name[:len(name)-1] + soft
	case strings.HasSuffix(name, "я"):
		return name[:len(name)-1] + hard
	}
	return name
}
