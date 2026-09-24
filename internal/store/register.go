package store

import (
	"strings"
	"unicode"

	"vkemu/internal/model"
)

func NormalizeLogin(login string) string {
	login = strings.TrimSpace(strings.ToLower(login))
	return strings.TrimPrefix(login, "@")
}

func ValidLogin(login string) bool {
	if len(login) < 3 || len(login) > 32 {
		return false
	}
	for _, r := range login {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '_', '.', '-', '+':
			continue
		}
		return false
	}
	return true
}

func (s *Store) LoginTaken(login string) bool {
	_, err := s.UserByDomain(NormalizeLogin(login))
	return err == nil
}

func (s *Store) Register(login, password, firstName, lastName string) (model.User, error) {
	login = NormalizeLogin(login)
	if !ValidLogin(login) {
		return model.User{}, ErrLoginInvalid
	}
	if s.LoginTaken(login) {
		return model.User{}, ErrLoginTaken
	}
	if firstName == "" {
		firstName = login
	}
	uid, err := s.NextUserID()
	if err != nil {
		return model.User{}, err
	}
	user := model.User{
		UID:       uid,
		FirstName: firstName,
		LastName:  lastName,
		Domain:    login,
		Password:  password,
		City:      1,
		Country:   1,
		Online:    true,
	}
	if err := s.CreateUser(user); err != nil {
		return model.User{}, err
	}
	return user, nil
}
