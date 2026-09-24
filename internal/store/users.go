package store

import (
	"database/sql"
	"errors"
	"strings"

	"vkemu/internal/model"
)

const userColumns = `uid, first_name, last_name, nickname, domain, sex, bdate, city, country,
	status, mobile_phone, home_phone, university_name, graduation, relation, online, password, avatar`

func (s *Store) CreateUser(u model.User) error {
	_, err := s.db.Exec(`INSERT INTO users(`+userColumns+`)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		u.UID, u.FirstName, u.LastName, u.Nickname, u.Domain, u.Sex, u.BDate, u.City, u.Country,
		u.Status, u.MobilePhone, u.HomePhone, u.UniversityName, u.Graduation, u.Relation, boolToInt(u.Online),
		u.Password, u.Avatar)
	return err
}

func (s *Store) UpsertUser(u model.User) error {
	_, err := s.db.Exec(`INSERT INTO users(`+userColumns+`)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(uid) DO UPDATE SET
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			nickname = excluded.nickname,
			domain = excluded.domain,
			sex = excluded.sex,
			bdate = excluded.bdate,
			city = excluded.city,
			country = excluded.country,
			status = excluded.status,
			mobile_phone = excluded.mobile_phone,
			home_phone = excluded.home_phone,
			university_name = excluded.university_name,
			graduation = excluded.graduation,
			relation = excluded.relation,
			online = excluded.online,
			password = excluded.password,
			avatar = excluded.avatar`,
		u.UID, u.FirstName, u.LastName, u.Nickname, u.Domain, u.Sex, u.BDate, u.City, u.Country,
		u.Status, u.MobilePhone, u.HomePhone, u.UniversityName, u.Graduation, u.Relation, boolToInt(u.Online),
		u.Password, u.Avatar)
	return err
}

func (s *Store) User(uid int) (model.User, error) {
	row := s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE uid = ?`, uid)
	return scanUser(row)
}

func (s *Store) UserByDomain(domain string) (model.User, error) {
	domain = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), "@")
	row := s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE domain = ? OR mobile_phone = ?`,
		domain, domain)
	return scanUser(row)
}

func (s *Store) Users(uids []int) ([]model.User, error) {
	if len(uids) == 0 {
		return nil, nil
	}
	query := `SELECT ` + userColumns + ` FROM users WHERE uid IN (` + placeholders(len(uids)) + `)`
	rows, err := s.db.Query(query, intsToAny(uids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectUsers(rows)
}

func (s *Store) AllUsers() ([]model.User, error) {
	rows, err := s.db.Query(`SELECT ` + userColumns + ` FROM users ORDER BY uid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectUsers(rows)
}

func (s *Store) SearchUsers(query string, exclude []int) ([]model.User, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	args := []any{pattern, pattern, pattern}
	cond := `(LOWER(first_name || ' ' || last_name) LIKE ? OR LOWER(domain) LIKE ? OR mobile_phone LIKE ?)`
	if len(exclude) > 0 {
		cond += ` AND uid NOT IN (` + placeholders(len(exclude)) + `)`
		args = append(args, intsToAny(exclude)...)
	}
	rows, err := s.db.Query(`SELECT `+userColumns+` FROM users WHERE `+cond+` ORDER BY last_name, first_name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectUsers(rows)
}

func (s *Store) UpdateUserStatus(uid int, status string) error {
	return s.exec(`UPDATE users SET status = ? WHERE uid = ?`, status, uid)
}

func (s *Store) SetUserAvatar(uid int, url string) error {
	return s.exec(`UPDATE users SET avatar = ? WHERE uid = ?`, url, uid)
}

func (s *Store) NextUserID() (int, error) {
	return s.nextID("user")
}

func (s *Store) MaxUserID() int {
	var uid int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(uid), 0) FROM users`).Scan(&uid); err != nil {
		return 0
	}
	return uid
}

func (s *Store) SetUserOnline(uid int, online bool) error {
	return s.exec(`UPDATE users SET online = ? WHERE uid = ?`, boolToInt(online), uid)
}

func (s *Store) OnlineFriendIDs(uid int) ([]int, error) {
	rows, err := s.db.Query(`SELECT f.fid FROM friends f JOIN users u ON u.uid = f.fid
		WHERE f.uid = ? AND u.online = 1 ORDER BY f.fid`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectInts(rows)
}

func scanUser(row interface{ Scan(...any) error }) (model.User, error) {
	var u model.User
	var online int
	err := row.Scan(&u.UID, &u.FirstName, &u.LastName, &u.Nickname, &u.Domain, &u.Sex, &u.BDate,
		&u.City, &u.Country, &u.Status, &u.MobilePhone, &u.HomePhone, &u.UniversityName,
		&u.Graduation, &u.Relation, &online, &u.Password, &u.Avatar)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, err
	}
	u.Online = online == 1
	return u, nil
}

func collectUsers(rows *sql.Rows) ([]model.User, error) {
	var out []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func collectInts(rows *sql.Rows) ([]int, error) {
	var out []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
