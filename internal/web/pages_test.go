package web_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"vkemu/internal/model"
)

func postFixture(ownerID, postID int, date int64) model.WallPost {
	return model.WallPost{
		OwnerID: ownerID,
		PostID:  postID,
		FromID:  ownerID,
		Date:    date,
		Text:    "Запись " + itoa(postID),
	}
}

func authClient(t *testing.T, base, login string) *http.Client {
	t.Helper()
	client := newClient(t)
	register(t, client, base, login)
	return client
}

func TestNewsfeedPaginationNoDuplicates(t *testing.T) {
	ts, db := newWeb(t)
	client := authClient(t, ts.URL, "tester")
	uid := db.MaxUserID()

	for i := 0; i < 25; i++ {
		post := postFixture(uid, i+1, db.Now()+int64(i))
		if err := db.CreatePost(post); err != nil {
			t.Fatalf("create post: %v", err)
		}
	}

	_, first := get(t, client, ts.URL+"/feed")
	firstIDs := postIDs(first)
	if len(firstIDs) != 20 {
		t.Fatalf("expected 20 posts on first page: %v", firstIDs)
	}

	index := strings.Index(first, "end_time=")
	if index < 0 {
		t.Fatalf("feed has no pagination link: %s", first)
	}
	endTime := first[index+len("end_time="):]
	if cut := strings.IndexAny(endTime, "\"&"); cut >= 0 {
		endTime = endTime[:cut]
	}

	_, second := get(t, client, ts.URL+"/feed?end_time="+endTime)
	secondIDs := postIDs(second)
	if len(secondIDs) != 5 {
		t.Fatalf("expected 5 posts on second page: %v", secondIDs)
	}
	for _, id := range secondIDs {
		for _, seen := range firstIDs {
			if id == seen {
				t.Fatalf("duplicate post %s across pages", id)
			}
		}
	}
}

func postIDs(page string) []string {
	var out []string
	for _, chunk := range strings.Split(page, "post/view?owner=") {
		index := strings.Index(chunk, "&amp;id=")
		if index < 0 {
			continue
		}
		rest := chunk[index+len("&amp;id="):]
		if cut := strings.IndexAny(rest, "\"<"); cut >= 0 {
			out = append(out, rest[:cut])
		}
	}
	return out
}

func TestProfileAndFriendsPages(t *testing.T) {
	ts, db := newWeb(t)
	alice := authClient(t, ts.URL, "alice")
	aliceUID := db.MaxUserID()
	bob := authClient(t, ts.URL, "bob")
	bobUID := db.MaxUserID()

	resp, body := get(t, alice, ts.URL+"/id/"+itoa(aliceUID))
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Тест Пользователь") {
		t.Fatalf("own profile: %d %s", resp.StatusCode, body)
	}

	resp, body = get(t, alice, ts.URL+"/id/"+itoa(bobUID))
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Добавить в друзья") {
		t.Fatalf("other profile: %d %s", resp.StatusCode, body)
	}

	add := postForm(t, alice, ts.URL+"/friend/add", url.Values{"uid": {itoa(bobUID)}})
	if add.StatusCode != http.StatusSeeOther {
		t.Fatalf("friend add: %d", add.StatusCode)
	}
	requests, err := db.FriendRequests(bobUID, 0, 10)
	if err != nil || len(requests) != 1 {
		t.Fatalf("requests: %v %v", requests, err)
	}

	_, list := get(t, bob, ts.URL+"/friends/requests")
	if !strings.Contains(list, "Принять") {
		t.Fatalf("requests page: %s", list)
	}
	accept := postForm(t, bob, ts.URL+"/friend/accept", url.Values{"uid": {itoa(aliceUID)}})
	if accept.StatusCode != http.StatusSeeOther {
		t.Fatalf("friend accept: %d", accept.StatusCode)
	}
	if !db.AreFriends(aliceUID, bobUID) {
		t.Fatalf("friendship not stored")
	}

	_, friends := get(t, alice, ts.URL+"/friends")
	if !strings.Contains(friends, "Тест Пользователь") {
		t.Fatalf("friends page: %s", friends)
	}
}

func TestMessagesFlow(t *testing.T) {
	ts, db := newWeb(t)
	alice := authClient(t, ts.URL, "alice")
	aliceUID := db.MaxUserID()
	bob := authClient(t, ts.URL, "bob")
	bobUID := db.MaxUserID()

	send := postForm(t, alice, ts.URL+"/messages/send", url.Values{
		"uid":     {itoa(bobUID)},
		"message": {"Привет"},
	})
	if send.StatusCode != http.StatusSeeOther {
		t.Fatalf("send status: %d", send.StatusCode)
	}

	_, inbox := get(t, bob, ts.URL+"/messages")
	if !strings.Contains(inbox, "Привет") {
		t.Fatalf("dialogs: %s", inbox)
	}
	_, conversation := get(t, bob, ts.URL+"/messages?uid="+itoa(aliceUID))
	if !strings.Contains(conversation, "Привет") {
		t.Fatalf("conversation: %s", conversation)
	}
	reply := postForm(t, bob, ts.URL+"/messages/send", url.Values{
		"uid":     {itoa(aliceUID)},
		"message": {"Ответ"},
	})
	if reply.StatusCode != http.StatusSeeOther {
		t.Fatalf("reply status: %d", reply.StatusCode)
	}
	messages, err := db.Messages(aliceUID, 0, 10)
	if err != nil || len(messages) != 2 {
		t.Fatalf("messages: %v %v", messages, err)
	}
}

func TestPhotoGalleryAndAlbum(t *testing.T) {
	ts, db := newWeb(t)
	client := authClient(t, ts.URL, "tester")
	uid := db.MaxUserID()

	album := postForm(t, client, ts.URL+"/albums/create", url.Values{"title": {"Отпуск"}})
	if album.StatusCode != http.StatusSeeOther {
		t.Fatalf("album status: %d", album.StatusCode)
	}
	albums, err := db.Albums(uid)
	if err != nil || len(albums) != 1 || albums[0].Title != "Отпуск" {
		t.Fatalf("albums: %v %v", albums, err)
	}

	body, contentType := multipartBody(t, map[string]string{"caption": "Море"}, "photo", "sea.jpg", []byte("jpeg-bytes"))
	resp, err := client.Post(ts.URL+"/photos/upload", contentType, body)
	if err != nil {
		t.Fatalf("photo upload: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("photo upload status %d: %s", resp.StatusCode, data)
	}

	_, gallery := get(t, client, ts.URL+"/photos")
	if !strings.Contains(gallery, "Море") || !strings.Contains(gallery, "/media/upload/") {
		t.Fatalf("gallery: %s", gallery)
	}
}

func TestVideoNoteAndDocUpload(t *testing.T) {
	ts, db := newWeb(t)
	client := authClient(t, ts.URL, "tester")
	uid := db.MaxUserID()

	videoBody, contentType := multipartBody(t, map[string]string{"title": "Клип"}, "file", "clip.mp4", []byte("mp4-bytes"))
	resp, err := client.Post(ts.URL+"/videos/upload", contentType, videoBody)
	if err != nil {
		t.Fatalf("video upload: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("video status: %d", resp.StatusCode)
	}
	videos, err := db.Videos(uid, 0, 10)
	if err != nil || len(videos) != 1 || videos[0].Title != "Клип" {
		t.Fatalf("videos: %v %v", videos, err)
	}

	note := postForm(t, client, ts.URL+"/notes/add", url.Values{"title": {"Список"}, "text": {"Молоко"}})
	if note.StatusCode != http.StatusSeeOther {
		t.Fatalf("note status: %d", note.StatusCode)
	}
	notes, err := db.Notes(uid, 0, 10)
	if err != nil || len(notes) != 1 || notes[0].Title != "Список" {
		t.Fatalf("notes: %v %v", notes, err)
	}

	docBody, contentType := multipartBody(t, nil, "file", "report.txt", []byte("hello"))
	docResp, err := client.Post(ts.URL+"/docs/upload", contentType, docBody)
	if err != nil {
		t.Fatalf("doc upload: %v", err)
	}
	defer docResp.Body.Close()
	if docResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("doc status: %d", docResp.StatusCode)
	}
	docs, err := db.Docs(uid, 0, 10)
	if err != nil || len(docs) != 1 || docs[0].Title != "report.txt" {
		t.Fatalf("docs: %v %v", docs, err)
	}
}

func TestAudioPostAttachment(t *testing.T) {
	ts, db := newWeb(t)
	client := authClient(t, ts.URL, "tester")
	uid := db.MaxUserID()

	body, contentType := multipartBody(t, map[string]string{"message": "Трек дня"}, "attachment", "song.mp3", []byte("mp3-bytes"))
	resp, err := client.Post(ts.URL+"/post", contentType, body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("post status %d: %s", resp.StatusCode, data)
	}
	posts, err := db.Posts(uid, 0, 10)
	if err != nil || len(posts) != 1 || posts[0].AttachType != "audio" {
		t.Fatalf("audio post: %v %v", posts, err)
	}

	_, feed := get(t, client, ts.URL+"/feed")
	if !strings.Contains(feed, "<audio controls") {
		t.Fatalf("feed without audio player: %s", feed)
	}
}

func TestSettingsAndStatusUpdate(t *testing.T) {
	ts, db := newWeb(t)
	client := authClient(t, ts.URL, "tester")
	uid := db.MaxUserID()

	resp := postForm(t, client, ts.URL+"/settings", url.Values{
		"first_name":      {"Иван"},
		"last_name":       {"Петров"},
		"nickname":        {"vanya"},
		"status":          {"на связи"},
		"sex":             {"2"},
		"bdate":           {"1.1.1990"},
		"city":            {"1"},
		"country":         {"1"},
		"university_name": {"МГУ"},
		"graduation":      {"2012"},
		"relation":        {"4"},
		"mobile_phone":    {"+7 900 000-00-00"},
		"home_phone":      {"+7 495 000-00-00"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("settings status: %d", resp.StatusCode)
	}
	user, err := db.User(uid)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if user.FirstName != "Иван" || user.Status != "на связи" || user.Nickname != "vanya" {
		t.Fatalf("user not updated: %+v", user)
	}
	if user.BDate != "1.1.1990" || user.City != 1 || user.Country != 1 || user.Relation != 4 {
		t.Fatalf("profile fields not stored: %+v", user)
	}
	if user.UniversityName != "МГУ" || user.Graduation != "2012" {
		t.Fatalf("education not stored: %+v", user)
	}
	if user.MobilePhone == "" || user.HomePhone == "" {
		t.Fatalf("phones not stored: %+v", user)
	}

	_, profile := get(t, client, ts.URL+"/id/"+itoa(uid))
	for _, expected := range []string{"День рождения: 1.1.1990", "Город: Москва", "МГУ", "женат"} {
		if !strings.Contains(profile, expected) {
			t.Fatalf("profile missing %q: %s", expected, profile)
		}
	}
}

func TestPlacesCheckinCreatesPost(t *testing.T) {
	ts, db := newWeb(t)
	client := authClient(t, ts.URL, "tester")
	uid := db.MaxUserID()

	resp := postForm(t, client, ts.URL+"/places/checkin", url.Values{
		"title":   {"Кафе"},
		"address": {"Ленина 1"},
		"text":    {"Кофе"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("checkin status: %d", resp.StatusCode)
	}
	posts, err := db.Posts(uid, 0, 10)
	if err != nil || len(posts) != 1 || posts[0].AttachType != "geo" {
		t.Fatalf("checkin post: %v %v", posts, err)
	}
	if !strings.Contains(posts[0].GeoJSON, "Кафе") {
		t.Fatalf("geo json: %s", posts[0].GeoJSON)
	}
}
