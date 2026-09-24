package web

import (
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"vkemu/internal/model"
)

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	uid := atoiOr(strings.TrimPrefix(r.URL.Path, "/id/"), v.user.UID)
	if uid == 0 {
		uid = v.user.UID
	}
	user, err := s.db.User(uid)
	if err != nil {
		s.fail(w, "Пользователь не найден")
		return
	}

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1(user.FullName()))
	if uid == v.user.UID {
		body.WriteString(s.avatarBlock(v))
	} else {
		body.WriteString(p("<img src=\"" + html.EscapeString(s.userPhoto(uid)) + "\" width=\"150\" height=\"150\" alt=\"avatar\">"))
	}
	if user.Status != "" {
		body.WriteString(p(html.EscapeString(user.Status)))
	}
	if uid == v.user.UID {
		body.WriteString(s.postForm())
	} else if s.db.AreFriends(v.user.UID, uid) {
		body.WriteString(postButton("/friend/delete", []string{hidden("uid", uid)}, "Удалить из друзей"))
		body.WriteString(p(link("/messages?uid="+strconv.Itoa(uid), "Написать сообщение")))
	} else {
		body.WriteString(postButton("/friend/add", []string{hidden("uid", uid)}, "Добавить в друзья"))
	}
	body.WriteString(s.profileStats(uid))

	body.WriteString(h2("Записи"))
	posts, err := s.db.Posts(uid, 0, 50)
	if err != nil {
		body.WriteString(p("Ошибка загрузки записей"))
	}
	if len(posts) == 0 {
		body.WriteString(p("Записей пока нет"))
	}
	for _, post := range posts {
		body.WriteString(s.postBlock(v, post))
	}
	s.page(w, user.FullName(), body.String())
}

func (s *Server) profileStats(uid int) string {
	user, err := s.db.User(uid)
	audios, _ := s.db.Audio(uid, 0, 1000, 0)
	friendIDs, _ := s.db.FriendIDs(uid)
	var body strings.Builder
	body.WriteString("<ul>")
	body.WriteString("<li>Друзей: " + strconv.Itoa(len(friendIDs)) + "</li>")
	body.WriteString("<li>Фото: " + strconv.Itoa(s.db.PhotoCountAll(uid)) + "</li>")
	body.WriteString("<li>Треков: " + strconv.Itoa(len(audios)) + "</li>")
	body.WriteString("</ul>")

	if err == nil {
		body.WriteString(h2("Информация"))
		body.WriteString("<ul>")
		if user.BDate != "" {
			body.WriteString("<li>День рождения: " + html.EscapeString(user.BDate) + "</li>")
		}
		if user.City != 0 {
			body.WriteString("<li>Город: " + html.EscapeString(s.cityName(user.City)) + "</li>")
		}
		if user.Country != 0 {
			body.WriteString("<li>Страна: " + html.EscapeString(s.countryName(user.Country)) + "</li>")
		}
		if user.UniversityName != "" {
			label := user.UniversityName
			if user.Graduation != "" {
				label += " (" + user.Graduation + ")"
			}
			body.WriteString("<li>Учился(ась): " + html.EscapeString(label) + "</li>")
		}
		if user.Relation > 0 {
			body.WriteString("<li>Семейное положение: " + html.EscapeString(relationLabel(user.Relation)) + "</li>")
		}
		if user.MobilePhone != "" {
			body.WriteString("<li>Мобильный: " + html.EscapeString(user.MobilePhone) + "</li>")
		}
		if user.HomePhone != "" {
			body.WriteString("<li>Домашний: " + html.EscapeString(user.HomePhone) + "</li>")
		}
		body.WriteString("</ul>")
	}
	return body.String()
}

func (s *Server) cityName(cid int) string {
	cities, err := s.db.Cities([]int{cid})
	if err != nil || len(cities) == 0 {
		return strconv.Itoa(cid)
	}
	return cities[0].Name
}

func (s *Server) countryName(cid int) string {
	countries, err := s.db.Countries([]int{cid})
	if err != nil || len(countries) == 0 {
		return strconv.Itoa(cid)
	}
	return countries[0].Name
}

func relationLabel(value int) string {
	for _, item := range relationOptions() {
		if item.value == strconv.Itoa(value) {
			return item.label
		}
	}
	return ""
}

func (s *Server) handleFriends(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	ids, err := s.db.FriendIDs(v.user.UID)

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Друзья"))
	body.WriteString(p(link("/friends/requests", "Заявки в друзья")))
	if err != nil {
		body.WriteString(p("Ошибка загрузки друзей"))
	}
	if len(ids) == 0 {
		body.WriteString(p("Друзей пока нет"))
	}
	users, _ := s.db.Users(ids)
	for _, user := range users {
		body.WriteString("<hr>")
		body.WriteString(p("<img src=\"" + html.EscapeString(s.userPhoto(user.UID)) + "\" width=\"48\" height=\"48\" alt=\"photo\"> " +
			link(profileURL(user.UID), user.FullName())))
		body.WriteString(p(link("/messages?uid="+strconv.Itoa(user.UID), "Написать сообщение")))
		body.WriteString(postButton("/friend/delete", []string{hidden("uid", user.UID)}, "Удалить"))
	}
	body.WriteString(s.userSearchForm())
	s.page(w, "Друзья", body.String())
}

func (s *Server) userSearchForm() string {
	var body strings.Builder
	body.WriteString(h2("Найти людей"))
	body.WriteString(`<form method="get" action="/search">`)
	body.WriteString(`<p>Имя или логин: <input type="text" name="q" value=""></p>`)
	body.WriteString(`<p><button type="submit">Найти</button></p>`)
	body.WriteString(`</form>`)
	return body.String()
}

func (s *Server) handleFriendRequests(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	requests, err := s.db.FriendRequests(v.user.UID, 0, 100)

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Заявки в друзья"))
	if err != nil {
		body.WriteString(p("Ошибка загрузки заявок"))
	}
	if len(requests) == 0 {
		body.WriteString(p("Заявок нет"))
	}
	for _, request := range requests {
		body.WriteString("<hr>")
		body.WriteString(p("<img src=\"" + html.EscapeString(s.userPhoto(request.FromUID)) + "\" width=\"48\" height=\"48\" alt=\"photo\"> " +
			link(profileURL(request.FromUID), s.name(request.FromUID))))
		body.WriteString(postButton("/friend/accept", []string{hidden("uid", request.FromUID)}, "Принять"))
		body.WriteString(postButton("/friend/delete", []string{hidden("uid", request.FromUID)}, "Отклонить"))
	}
	s.page(w, "Заявки", body.String())
}

func (s *Server) handleFriendAdd(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	uid := formUID(r)
	if uid != 0 && uid != v.user.UID && !s.db.AreFriends(v.user.UID, uid) {
		s.db.SaveFriendRequest(friendRequest(v.user.UID, uid, s.db.Now()))
	}
	s.redirectBack(w, r, "/friends")
}

func (s *Server) handleFriendAccept(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	uid := formUID(r)
	if uid != 0 {
		s.db.UpsertFriend(v.user.UID, uid)
		s.db.UpsertFriend(uid, v.user.UID)
		s.db.DeleteFriendRequest(uid, v.user.UID)
	}
	http.Redirect(w, r, "/friends/requests", http.StatusSeeOther)
}

func (s *Server) handleFriendDelete(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	uid := formUID(r)
	if uid != 0 {
		s.db.DeleteFriend(v.user.UID, uid)
	}
	s.redirectBack(w, r, "/friends")
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Поиск людей"))
	body.WriteString(`<form method="get" action="/search">`)
	body.WriteString(`<p>Имя или логин: <input type="text" name="q" value="` + valueAttr(query) + `"></p>`)
	body.WriteString(`<p><button type="submit">Найти</button></p>`)
	body.WriteString(`</form>`)

	if query != "" {
		users, err := s.db.SearchUsers(query, []int{v.user.UID})
		if err != nil {
			body.WriteString(p("Ошибка поиска"))
		}
		if len(users) == 0 {
			body.WriteString(p("Никого не найдено"))
		}
		for _, user := range users {
			body.WriteString("<hr>")
			body.WriteString(p("<img src=\"" + html.EscapeString(s.userPhoto(user.UID)) + "\" width=\"48\" height=\"48\" alt=\"photo\"> " +
				link(profileURL(user.UID), user.FullName())))
			if s.db.AreFriends(v.user.UID, user.UID) {
				body.WriteString(p("Уже в друзьях"))
				continue
			}
			body.WriteString(postButton("/friend/add", []string{hidden("uid", user.UID)}, "Добавить в друзья"))
		}
	}
	s.page(w, "Поиск", body.String())
}

func formUID(r *http.Request) int {
	if err := r.ParseForm(); err != nil {
		return 0
	}
	return atoiOr(r.FormValue("uid"), 0)
}

func (s *Server) redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	if back := r.FormValue("back"); strings.HasPrefix(back, "/") {
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	referer := r.Header.Get("Referer")
	if parsed, err := url.Parse(referer); err == nil && parsed.Path != "" && strings.HasPrefix(parsed.Path, "/") {
		target := parsed.Path
		if parsed.RawQuery != "" {
			target += "?" + parsed.RawQuery
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fallback, http.StatusSeeOther)
}

func friendRequest(fromUID, toUID int, date int64) model.FriendRequest {
	return model.FriendRequest{FromUID: fromUID, ToUID: toUID, Date: date}
}
