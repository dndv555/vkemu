package web

import (
	"net/http"
	"strings"

	"vkemu/internal/model"
	"vkemu/internal/store"
)

type Server struct {
	db        *store.Store
	uploadDir string
	sessions  *sessionStore
}

func New(db *store.Store, uploadDir string) *Server {
	return &Server{db: db, uploadDir: uploadDir, sessions: newSessionStore()}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/register", s.handleRegister)
	mux.HandleFunc("/logout", s.handleLogout)

	mux.HandleFunc("/feed", s.handleFeed)
	mux.HandleFunc("/id/", s.handleProfile)
	mux.HandleFunc("/post", s.handlePost)
	mux.HandleFunc("/post/view", s.handlePostView)
	mux.HandleFunc("/post/delete", s.handlePostDelete)
	mux.HandleFunc("/like", s.handleLike)
	mux.HandleFunc("/comment", s.handleComment)

	mux.HandleFunc("/friends", s.handleFriends)
	mux.HandleFunc("/friends/requests", s.handleFriendRequests)
	mux.HandleFunc("/friend/add", s.handleFriendAdd)
	mux.HandleFunc("/friend/accept", s.handleFriendAccept)
	mux.HandleFunc("/friend/delete", s.handleFriendDelete)

	mux.HandleFunc("/messages", s.handleMessages)
	mux.HandleFunc("/messages/send", s.handleMessageSend)
	mux.HandleFunc("/messages/delete", s.handleMessageDelete)

	mux.HandleFunc("/photos", s.handlePhotos)
	mux.HandleFunc("/photos/upload", s.handlePhotoUpload)
	mux.HandleFunc("/photos/delete", s.handlePhotoDelete)
	mux.HandleFunc("/albums/create", s.handleAlbumCreate)

	mux.HandleFunc("/music", s.handleMusic)
	mux.HandleFunc("/music/upload", s.handleMusicUpload)
	mux.HandleFunc("/music/delete", s.handleMusicDelete)

	mux.HandleFunc("/videos", s.handleVideos)
	mux.HandleFunc("/videos/upload", s.handleVideoUpload)

	mux.HandleFunc("/notes", s.handleNotes)
	mux.HandleFunc("/notes/add", s.handleNoteAdd)
	mux.HandleFunc("/notes/delete", s.handleNoteDelete)

	mux.HandleFunc("/docs", s.handleDocs)
	mux.HandleFunc("/docs/upload", s.handleDocUpload)
	mux.HandleFunc("/docs/delete", s.handleDocDelete)

	mux.HandleFunc("/places", s.handlePlaces)
	mux.HandleFunc("/places/checkin", s.handleCheckin)

	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/settings", s.handleSettings)
	mux.HandleFunc("/status", s.handleStatus)

	mux.HandleFunc("/upload-avatar", s.handleUploadAvatar)
	mux.HandleFunc("/media/", s.handleMedia)
	return mux
}

type viewer struct {
	user model.User
}

func (s *Server) current(r *http.Request) (viewer, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return viewer{}, false
	}
	uid, ok := s.sessions.uid(cookie.Value)
	if !ok {
		return viewer{}, false
	}
	user, err := s.db.User(uid)
	if err != nil {
		return viewer{}, false
	}
	return viewer{user: user}, true
}

func (s *Server) require(w http.ResponseWriter, r *http.Request) (viewer, bool) {
	v, ok := s.current(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return viewer{}, false
	}
	return v, true
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if _, ok := s.current(r); ok {
		http.Redirect(w, r, "/feed", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.page(w, "Вход", s.loginForm("", ""))
		return
	}
	if err := r.ParseForm(); err != nil {
		s.page(w, "Вход", s.loginForm("", "Некорректная форма"))
		return
	}
	login := store.NormalizeLogin(r.FormValue("login"))
	password := r.FormValue("password")

	user, err := s.db.UserByDomain(login)
	if err != nil || (user.Password != "" && user.Password != password) {
		s.page(w, "Вход", s.loginForm(login, "Неверный логин или пароль"))
		return
	}
	s.startSession(w, user.UID)
	http.Redirect(w, r, "/feed", http.StatusSeeOther)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.page(w, "Регистрация", s.registerForm("", "", ""))
		return
	}
	if err := r.ParseForm(); err != nil {
		s.page(w, "Регистрация", s.registerForm("", "", "Некорректная форма"))
		return
	}
	login := store.NormalizeLogin(r.FormValue("login"))
	password := r.FormValue("password")
	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))

	if !store.ValidLogin(login) {
		s.page(w, "Регистрация", s.registerForm(login, firstName, "Логин: 3-32 символа, латиница, цифры, _ . - +"))
		return
	}
	if password == "" {
		s.page(w, "Регистрация", s.registerForm(login, firstName, "Введите пароль"))
		return
	}
	user, err := s.db.Register(login, password, firstName, lastName)
	if err != nil {
		message := "Не удалось зарегистрироваться"
		if err == store.ErrLoginTaken {
			message = "Такой логин уже занят"
		}
		s.page(w, "Регистрация", s.registerForm(login, firstName, message))
		return
	}
	s.startSession(w, user.UID)
	http.Redirect(w, r, "/feed", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		s.sessions.drop(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) startSession(w http.ResponseWriter, uid int) {
	token := s.sessions.create(uid)
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/"})
}

func (s *Server) loginForm(login, message string) string {
	var body strings.Builder
	body.WriteString(h1("Вход"))
	body.WriteString(errorBox(message))
	body.WriteString(`<form method="post" action="/login">`)
	body.WriteString(textInput("login", "Логин", login))
	body.WriteString(`<p>Пароль: <input type="password" name="password" value=""></p>`)
	body.WriteString(`<p><button type="submit">Войти</button></p>`)
	body.WriteString(`</form>`)
	body.WriteString(p(link("/register", "Регистрация")))
	return body.String()
}

func (s *Server) registerForm(login, firstName, message string) string {
	var body strings.Builder
	body.WriteString(h1("Регистрация"))
	body.WriteString(errorBox(message))
	body.WriteString(`<form method="post" action="/register">`)
	body.WriteString(textInput("login", "Логин", login))
	body.WriteString(`<p>Пароль: <input type="password" name="password" value=""></p>`)
	body.WriteString(textInput("first_name", "Имя", firstName))
	body.WriteString(`<p>Фамилия: <input type="text" name="last_name" value=""></p>`)
	body.WriteString(`<p><button type="submit">Создать аккаунт</button></p>`)
	body.WriteString(`</form>`)
	body.WriteString(p(link("/login", "У меня есть аккаунт")))
	return body.String()
}
