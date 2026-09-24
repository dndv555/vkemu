package web

import (
	"html"
	"net/http"
	"strconv"
	"strings"

	"vkemu/internal/model"
	"vkemu/internal/upload"
)

func (s *Server) handlePlaces(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	places, err := s.db.SearchPlaces(0, 0, 20000)

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Места"))
	body.WriteString(h2("Добавить место"))
	body.WriteString(`<form method="post" action="/places/checkin">`)
	body.WriteString(`<p>Название: <input type="text" name="title" value=""></p>`)
	body.WriteString(`<p>Адрес: <input type="text" name="address" value=""></p>`)
	body.WriteString(`<p>Заметка: <input type="text" name="text" value=""></p>`)
	body.WriteString(`<p><button type="submit">Отметиться</button></p>`)
	body.WriteString(`</form>`)

	body.WriteString(h2("Все места"))
	if err != nil {
		body.WriteString(p("Ошибка загрузки мест"))
	}
	if len(places) == 0 {
		body.WriteString(p("Мест пока нет"))
	}
	for _, place := range places {
		body.WriteString("<hr>")
		body.WriteString(p("<b>" + html.EscapeString(place.Title) + "</b> — " + html.EscapeString(place.Address)))
		body.WriteString(p("Отметок: " + strconv.Itoa(place.Checkins)))
		body.WriteString(postButton("/places/checkin", []string{
			hidden("place_id", place.PlaceID),
			hiddenText("text", ""),
		}, "Отметиться"))
	}
	s.page(w, "Места", body.String())
}

func (s *Server) handleCheckin(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, "Некорректная форма")
		return
	}
	placeID := atoiOr(r.FormValue("place_id"), 0)
	text := strings.TrimSpace(r.FormValue("text"))
	if placeID == 0 {
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Redirect(w, r, "/places", http.StatusSeeOther)
			return
		}
		newID, err := s.db.NextPlaceID()
		if err != nil {
			s.fail(w, "Не удалось создать место")
			return
		}
		place := model.Place{
			PlaceID:   newID,
			Title:     title,
			Address:   strings.TrimSpace(r.FormValue("address")),
			Type:      1,
			CityID:    1,
			CountryID: 1,
		}
		if err := s.db.AddPlace(place); err != nil {
			s.fail(w, "Не удалось создать место")
			return
		}
		placeID = newID
	}
	place, err := s.db.Place(placeID)
	if err != nil {
		s.fail(w, "Место не найдено")
		return
	}
	checkin := model.Checkin{
		UID:       v.user.UID,
		PlaceID:   placeID,
		Date:      s.db.Now(),
		Text:      text,
		Latitude:  place.Latitude,
		Longitude: place.Longitude,
	}
	if err := s.db.AddCheckin(checkin); err != nil {
		s.fail(w, "Не удалось отметиться")
		return
	}
	s.createCheckinPost(v.user.UID, place, text)
	http.Redirect(w, r, "/feed", http.StatusSeeOther)
}

func (s *Server) createCheckinPost(uid int, place model.Place, text string) {
	postID, err := s.db.NextPostID(uid)
	if err != nil {
		return
	}
	post := model.WallPost{
		OwnerID:    uid,
		PostID:     postID,
		FromID:     uid,
		Date:       s.db.Now(),
		Text:       text,
		GeoJSON:    `{"place":{"place_id":` + strconv.Itoa(place.PlaceID) + `,"title":` + quoteJSON(place.Title) + `}}`,
		AttachType: "geo",
	}
	s.db.CreatePost(post)
}

func quoteJSON(value string) string {
	return strconv.Quote(value)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			s.fail(w, "Некорректная форма")
			return
		}
		if err := s.applySettings(r, &v.user); err != nil {
			s.fail(w, "Не удалось сохранить профиль")
			return
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	var body strings.Builder
	body.WriteString(s.nav(v))
	body.WriteString(h1("Настройки"))
	body.WriteString(s.settingsForm(v.user))
	body.WriteString(s.avatarBlock(v))
	body.WriteString(p("<a href=\"/logout\">Выйти</a>"))
	s.page(w, "Настройки", body.String())
}

func (s *Server) applySettings(r *http.Request, user *model.User) error {
	user.FirstName = strings.TrimSpace(r.FormValue("first_name"))
	user.LastName = strings.TrimSpace(r.FormValue("last_name"))
	user.Nickname = strings.TrimSpace(r.FormValue("nickname"))
	user.Status = strings.TrimSpace(r.FormValue("status"))
	user.Sex = atoiOr(r.FormValue("sex"), user.Sex)
	user.BDate = strings.TrimSpace(r.FormValue("bdate"))
	user.City = atoiOr(r.FormValue("city"), user.City)
	user.Country = atoiOr(r.FormValue("country"), user.Country)
	user.UniversityName = strings.TrimSpace(r.FormValue("university_name"))
	user.Graduation = strings.TrimSpace(r.FormValue("graduation"))
	user.Relation = atoiOr(r.FormValue("relation"), user.Relation)
	user.MobilePhone = strings.TrimSpace(r.FormValue("mobile_phone"))
	user.HomePhone = strings.TrimSpace(r.FormValue("home_phone"))
	return s.db.UpsertUser(*user)
}

func (s *Server) settingsForm(user model.User) string {
	var body strings.Builder
	body.WriteString(h2("Профиль"))
	body.WriteString(`<form method="post" action="/settings">`)
	body.WriteString(textInput("first_name", "Имя", user.FirstName))
	body.WriteString(textInput("last_name", "Фамилия", user.LastName))
	body.WriteString(textInput("nickname", "Никнейм", user.Nickname))
	body.WriteString(`<p>Пол: ` + selectField("sex", []option{
		{"0", "не указан"}, {"2", "мужской"}, {"1", "женский"},
	}, strconv.Itoa(user.Sex)) + `</p>`)
	body.WriteString(textInput("bdate", "Дата рождения (Д.М.ГГГГ)", user.BDate))
	body.WriteString(`<p>Город: ` + s.citySelect(user.City) + `</p>`)
	body.WriteString(`<p>Страна: ` + s.countrySelect(user.Country) + `</p>`)
	body.WriteString(textInput("university_name", "Учебное заведение", user.UniversityName))
	body.WriteString(textInput("graduation", "Год окончания", user.Graduation))
	body.WriteString(`<p>Семейное положение: ` + selectField("relation", relationOptions(), strconv.Itoa(user.Relation)) + `</p>`)
	body.WriteString(textInput("mobile_phone", "Мобильный телефон", user.MobilePhone))
	body.WriteString(textInput("home_phone", "Домашний телефон", user.HomePhone))
	body.WriteString(textInput("status", "Статус", user.Status))
	body.WriteString(`<p><button type="submit">Сохранить</button></p>`)
	body.WriteString(`</form>`)
	return body.String()
}

func (s *Server) citySelect(current int) string {
	cities, err := s.db.Cities([]int{1, 2, 3, 4, 5})
	if err != nil || len(cities) == 0 {
		return textInputInline("city", strconv.Itoa(current))
	}
	options := make([]option, 0, len(cities)+1)
	options = append(options, option{"0", "не указан"})
	for _, city := range cities {
		options = append(options, option{strconv.Itoa(city.CID), city.Name})
	}
	return selectField("city", options, strconv.Itoa(current))
}

func (s *Server) countrySelect(current int) string {
	countries, err := s.db.Countries([]int{1, 2, 3})
	if err != nil || len(countries) == 0 {
		return textInputInline("country", strconv.Itoa(current))
	}
	options := make([]option, 0, len(countries)+1)
	options = append(options, option{"0", "не указана"})
	for _, country := range countries {
		options = append(options, option{strconv.Itoa(country.CID), country.Name})
	}
	return selectField("country", options, strconv.Itoa(current))
}

func relationOptions() []option {
	return []option{
		{"0", "не указано"},
		{"1", "не женат/не замужем"},
		{"2", "есть друг/подруга"},
		{"3", "помолвлен/помолвлена"},
		{"4", "женат/замужем"},
		{"5", "всё сложно"},
		{"6", "в активном поиске"},
		{"7", "влюблён/влюблена"},
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, "Некорректная форма")
		return
	}
	s.db.UpdateUserStatus(v.user.UID, strings.TrimSpace(r.FormValue("text")))
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (s *Server) handleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	v, ok := s.require(w, r)
	if !ok {
		return
	}
	part, err := upload.ReadFile(r)
	if err != nil {
		s.fail(w, "Не удалось прочитать файл")
		return
	}
	item, err := s.saveUpload("avatar", v.user.UID, part)
	if err != nil {
		s.fail(w, "Не удалось сохранить аватар")
		return
	}
	if err := s.db.SetUserAvatar(v.user.UID, upload.URL(item)); err != nil {
		s.fail(w, "Не удалось обновить аватар")
		return
	}
	s.redirectBack(w, r, "/settings")
}
