package web_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"vkemu/internal/store"
	"vkemu/internal/web"
)

func newWeb(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := db.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	srv := web.New(db, filepath.Join(dir, "uploads"))
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		db.Close()
	})
	return ts, db
}

func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client
}

func postForm(t *testing.T, client *http.Client, url string, form url.Values) *http.Response {
	t.Helper()
	resp, err := client.PostForm(url, form)
	if err != nil {
		t.Fatalf("post form: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func get(t *testing.T, client *http.Client, url string) (*http.Response, string) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return resp, string(body)
}

func multipartBody(t *testing.T, fields map[string]string, fileField, filename string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	if fileField != "" {
		part, err := writer.CreateFormFile(fileField, filename)
		if err != nil {
			t.Fatalf("create file: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return &buf, writer.FormDataContentType()
}

func register(t *testing.T, client *http.Client, base, login string) {
	t.Helper()
	resp := postForm(t, client, base+"/register", url.Values{
		"login":      {login},
		"password":   {"secret"},
		"first_name": {"Тест"},
		"last_name":  {"Пользователь"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("register status %d: %s", resp.StatusCode, body)
	}
	if location := resp.Header.Get("Location"); location != "/feed" {
		t.Fatalf("unexpected redirect: %s", location)
	}
}

func TestRegisterAndFeedAccess(t *testing.T) {
	ts, _ := newWeb(t)
	client := newClient(t)

	resp, _ := get(t, client, ts.URL+"/")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("index redirect: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}

	register(t, client, ts.URL, "tester")
	resp, body := get(t, client, ts.URL+"/feed")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("feed status: %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Моя страница") || !strings.Contains(body, "Новая запись") {
		t.Fatalf("feed body: %s", body)
	}
}

func TestRegisterRejectsDuplicateLogin(t *testing.T) {
	ts, _ := newWeb(t)
	client := newClient(t)
	register(t, client, ts.URL, "tester")

	other := newClient(t)
	resp := postForm(t, other, ts.URL+"/register", url.Values{
		"login":    {"tester"},
		"password": {"secret"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("duplicate status: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "занят") {
		t.Fatalf("duplicate body: %s", body)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	ts, _ := newWeb(t)
	register(t, newClient(t), ts.URL, "tester")

	client := newClient(t)
	resp := postForm(t, client, ts.URL+"/login", url.Values{
		"login":    {"tester"},
		"password": {"wrong"},
	})
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Неверный логин") {
		t.Fatalf("login body: %s", body)
	}

	ok := postForm(t, client, ts.URL+"/login", url.Values{
		"login":    {"tester"},
		"password": {"secret"},
	})
	if ok.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status: %d", ok.StatusCode)
	}
}

func TestAvatarUpload(t *testing.T) {
	ts, db := newWeb(t)
	client := newClient(t)
	register(t, client, ts.URL, "tester")

	body, contentType := multipartBody(t, nil, "file", "avatar.png", []byte("png-bytes"))
	resp, err := client.Post(ts.URL+"/upload-avatar", contentType, body)
	if err != nil {
		t.Fatalf("upload avatar: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("avatar status %d: %s", resp.StatusCode, data)
	}

	uid := db.MaxUserID()
	user, err := db.User(uid)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if !strings.HasPrefix(user.Avatar, "/media/upload/") {
		t.Fatalf("avatar not stored: %q", user.Avatar)
	}

	_, feed := get(t, client, ts.URL+"/feed")
	if !strings.Contains(feed, user.Avatar) {
		t.Fatalf("avatar not rendered: %s", feed)
	}
}

func TestPhotoPostAndMusicUpload(t *testing.T) {
	ts, db := newWeb(t)
	client := newClient(t)
	register(t, client, ts.URL, "tester")
	uid := db.MaxUserID()

	body, contentType := multipartBody(t, map[string]string{"message": "Пост с фото"}, "photo", "photo.jpg", []byte("jpeg-bytes"))
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
	if err != nil {
		t.Fatalf("posts: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected one post: %v", posts)
	}
	if posts[0].AttachType != "photo" {
		t.Fatalf("post without photo attachment: %v", posts[0])
	}

	_, feed := get(t, client, ts.URL+"/feed")
	if !strings.Contains(feed, "Пост с фото") || !strings.Contains(feed, "<img src=\"/media/upload/") {
		t.Fatalf("feed missing photo post: %s", feed)
	}

	music, contentType := multipartBody(t, map[string]string{"artist": "Кино", "title": "Звезда"}, "file", "track.mp3", []byte("mp3-bytes"))
	musicResp, err := client.Post(ts.URL+"/music/upload", contentType, music)
	if err != nil {
		t.Fatalf("music: %v", err)
	}
	defer musicResp.Body.Close()
	if musicResp.StatusCode != http.StatusSeeOther {
		data, _ := io.ReadAll(musicResp.Body)
		t.Fatalf("music status %d: %s", musicResp.StatusCode, data)
	}

	tracks, err := db.Audio(uid, 0, 10, 0)
	if err != nil {
		t.Fatalf("audio: %v", err)
	}
	if len(tracks) != 1 || tracks[0].Artist != "Кино" || tracks[0].Title != "Звезда" {
		t.Fatalf("audio not saved: %v", tracks)
	}

	_, page := get(t, client, ts.URL+"/music")
	if !strings.Contains(page, "Кино — Звезда") || !strings.Contains(page, "<audio") {
		t.Fatalf("music page: %s", page)
	}
}

func TestLogoutClearsSession(t *testing.T) {
	ts, _ := newWeb(t)
	client := newClient(t)
	register(t, client, ts.URL, "tester")

	resp := postForm(t, client, ts.URL+"/logout", url.Values{})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("logout status: %d", resp.StatusCode)
	}
	feed, _ := get(t, client, ts.URL+"/feed")
	if feed.StatusCode != http.StatusSeeOther || feed.Header.Get("Location") != "/login" {
		t.Fatalf("feed after logout: %d %s", feed.StatusCode, feed.Header.Get("Location"))
	}
}

func TestPostViewLikesAndComments(t *testing.T) {
	ts, db := newWeb(t)
	client := newClient(t)
	register(t, client, ts.URL, "tester")
	uid := db.MaxUserID()

	body, contentType := multipartBody(t, map[string]string{"message": "Пост с фото"}, "photo", "photo.jpg", []byte("jpeg-bytes"))
	resp, err := client.Post(ts.URL+"/post", contentType, body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("post status %d", resp.StatusCode)
	}
	posts, err := db.Posts(uid, 0, 10)
	if err != nil || len(posts) != 1 {
		t.Fatalf("expected one post: %v %v", posts, err)
	}
	post := posts[0]

	page, html := get(t, client, ts.URL+"/post/view?owner="+itoa(uid)+"&id="+itoa(post.PostID))
	if page.StatusCode != http.StatusOK {
		t.Fatalf("post view status: %d", page.StatusCode)
	}
	if !strings.Contains(html, "Пост с фото") || !strings.Contains(html, "Комментарии") {
		t.Fatalf("post view body: %s", html)
	}

	like := postForm(t, client, ts.URL+"/like", url.Values{
		"owner_id": {itoa(uid)},
		"post_id":  {itoa(post.PostID)},
		"action":   {"add"},
	})
	if like.StatusCode != http.StatusSeeOther {
		t.Fatalf("like status: %d", like.StatusCode)
	}
	if !db.IsLiked("post", uid, post.PostID, uid) {
		t.Fatalf("like not stored")
	}

	comment := postForm(t, client, ts.URL+"/comment", url.Values{
		"owner_id": {itoa(uid)},
		"post_id":  {itoa(post.PostID)},
		"text":     {"Мой комментарий"},
	})
	if comment.StatusCode != http.StatusSeeOther {
		t.Fatalf("comment status: %d", comment.StatusCode)
	}
	comments, err := db.Comments("post", uid, post.PostID, 0, 10)
	if err != nil || len(comments) != 1 {
		t.Fatalf("comments not stored: %v %v", comments, err)
	}

	updated, updatedHTML := get(t, client, ts.URL+"/post/view?owner="+itoa(uid)+"&id="+itoa(post.PostID))
	if updated.StatusCode != http.StatusOK {
		t.Fatalf("updated post view status: %d", updated.StatusCode)
	}
	if !strings.Contains(updatedHTML, "Мой комментарий") || !strings.Contains(updatedHTML, "Убрать лайк") {
		t.Fatalf("post view not updated: %s", updatedHTML)
	}
}

func TestUploadURLKeepsExtension(t *testing.T) {
	ts, db := newWeb(t)
	client := newClient(t)
	register(t, client, ts.URL, "tester")
	uid := db.MaxUserID()

	body, contentType := multipartBody(t, nil, "file", "avatar.png", []byte("png-bytes"))
	resp, err := client.Post(ts.URL+"/upload-avatar", contentType, body)
	if err != nil {
		t.Fatalf("upload avatar: %v", err)
	}
	defer resp.Body.Close()

	user, err := db.User(uid)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if !strings.HasSuffix(user.Avatar, ".png") {
		t.Fatalf("avatar lost extension: %q", user.Avatar)
	}
	mediaResp, _ := get(t, client, ts.URL+user.Avatar)
	if mediaResp.StatusCode != http.StatusOK {
		t.Fatalf("avatar media status: %d", mediaResp.StatusCode)
	}
	if mediaResp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("avatar content type: %s", mediaResp.Header.Get("Content-Type"))
	}
}

func TestAudioSupportsRangeRequests(t *testing.T) {
	ts, _ := newWeb(t)
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/media/audio/1/1.wav", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Range", "bytes=0-99")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("range request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("range status: %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Range") == "" || resp.Header.Get("Content-Type") != "audio/wav" {
		t.Fatalf("range headers: %v", resp.Header)
	}
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
