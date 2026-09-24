package web_test

import (
	"net/url"
	"strings"
	"testing"
)

func TestMessageFriendPickerAndDirectLink(t *testing.T) {
	ts, db := newWeb(t)
	alice := authClient(t, ts.URL, "alice")
	aliceUID := db.MaxUserID()
	bob := authClient(t, ts.URL, "bob")
	bobUID := db.MaxUserID()

	_, empty := get(t, alice, ts.URL+"/messages")
	if !strings.Contains(empty, "Сначала добавьте друзей") {
		t.Fatalf("expected friend hint without friends: %s", empty)
	}

	if err := db.UpsertFriend(aliceUID, bobUID); err != nil {
		t.Fatalf("add friend: %v", err)
	}
	if err := db.UpsertFriend(bobUID, aliceUID); err != nil {
		t.Fatalf("add friend: %v", err)
	}

	_, picker := get(t, alice, ts.URL+"/messages")
	if !strings.Contains(picker, "<select name=\"uid\">") || !strings.Contains(picker, "Тест Пользователь") {
		t.Fatalf("friend picker missing: %s", picker)
	}

	_, profile := get(t, alice, ts.URL+"/id/"+itoa(bobUID))
	if !strings.Contains(profile, "/messages?uid="+itoa(bobUID)) {
		t.Fatalf("profile missing message link: %s", profile)
	}
	_, friends := get(t, alice, ts.URL+"/friends")
	if !strings.Contains(friends, "/messages?uid="+itoa(bobUID)) {
		t.Fatalf("friends page missing message link: %s", friends)
	}

	send := postForm(t, alice, ts.URL+"/messages/send", url.Values{
		"uid":     {itoa(bobUID)},
		"message": {"Привет из веба"},
	})
	if send.StatusCode != 303 {
		t.Fatalf("send status: %d", send.StatusCode)
	}
	if location := send.Header.Get("Location"); location != "/messages?uid="+itoa(bobUID) {
		t.Fatalf("send redirect: %s", location)
	}

	_, conversation := get(t, bob, ts.URL+"/messages?uid="+itoa(aliceUID))
	if !strings.Contains(conversation, "Привет из веба") {
		t.Fatalf("conversation missing message: %s", conversation)
	}

	messages, err := db.Conversation(aliceUID, bobUID, 0, 10)
	if err != nil || len(messages) != 1 {
		t.Fatalf("conversation not stored: %v %v", messages, err)
	}
	if messages[0].FromID != aliceUID || messages[0].ToID != bobUID {
		t.Fatalf("message fields: %+v", messages[0])
	}
	if messages[0].ChatID != 0 {
		t.Fatalf("personal message should not be a chat: %+v", messages[0])
	}
}
