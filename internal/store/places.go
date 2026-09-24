package store

import (
	"database/sql"

	"vkemu/internal/model"
)

func (s *Store) AddCity(city model.City) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO cities(cid, name) VALUES(?,?)`, city.CID, city.Name)
	return err
}

func (s *Store) Cities(ids []int) ([]model.City, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT cid, name FROM cities WHERE cid IN (`+placeholders(len(ids))+`)`,
		intsToAny(ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.City
	for rows.Next() {
		var c model.City
		if err := rows.Scan(&c.CID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) AddCountry(country model.Country) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO countries(cid, name) VALUES(?,?)`, country.CID, country.Name)
	return err
}

func (s *Store) Countries(ids []int) ([]model.Country, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT cid, name FROM countries WHERE cid IN (`+placeholders(len(ids))+`)`,
		intsToAny(ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Country
	for rows.Next() {
		var c model.Country
		if err := rows.Scan(&c.CID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) AddPlace(place model.Place) error {
	_, err := s.db.Exec(`INSERT INTO places(place_id, title, address, latitude, longitude, type, checkins, city_id, country_id)
		VALUES(?,?,?,?,?,?,?,?,?)
		ON CONFLICT(place_id) DO UPDATE SET title = excluded.title, address = excluded.address,
			latitude = excluded.latitude, longitude = excluded.longitude, type = excluded.type`,
		place.PlaceID, place.Title, place.Address, place.Latitude, place.Longitude, place.Type,
		place.Checkins, place.CityID, place.CountryID)
	return err
}

func (s *Store) NextPlaceID() (int, error) {
	return s.nextID("place")
}

func (s *Store) Place(placeID int) (model.Place, error) {
	var p model.Place
	err := s.db.QueryRow(`SELECT place_id, title, address, latitude, longitude, type, checkins, city_id, country_id
		FROM places WHERE place_id = ?`, placeID).
		Scan(&p.PlaceID, &p.Title, &p.Address, &p.Latitude, &p.Longitude, &p.Type, &p.Checkins,
			&p.CityID, &p.CountryID)
	if err == sql.ErrNoRows {
		return model.Place{}, ErrNotFound
	}
	if err != nil {
		return model.Place{}, err
	}
	return p, nil
}

func (s *Store) Places(ids []int) ([]model.Place, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT place_id, title, address, latitude, longitude, type, checkins, city_id, country_id
		FROM places WHERE place_id IN (`+placeholders(len(ids))+`)`, intsToAny(ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Place
	for rows.Next() {
		var p model.Place
		if err := rows.Scan(&p.PlaceID, &p.Title, &p.Address, &p.Latitude, &p.Longitude, &p.Type,
			&p.Checkins, &p.CityID, &p.CountryID); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) SearchPlaces(lat, lon float64, radius int) ([]model.Place, error) {
	rows, err := s.db.Query(`SELECT place_id, title, address, latitude, longitude, type, checkins, city_id, country_id
		FROM places`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	places, err := collectPlaces(rows)
	if err != nil {
		return nil, err
	}
	out := make([]model.Place, 0, len(places))
	for _, p := range places {
		if distance(lat, lon, p.Latitude, p.Longitude) <= float64(radius)*1000 {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Store) AddCheckin(checkin model.Checkin) error {
	id, err := s.nextID("checkin")
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`INSERT INTO checkins(id, uid, place_id, date, text, latitude, longitude)
		VALUES(?,?,?,?,?,?,?)`, id, checkin.UID, checkin.PlaceID, checkin.Date, checkin.Text,
		checkin.Latitude, checkin.Longitude); err != nil {
		return err
	}
	return s.exec(`UPDATE places SET checkins = checkins + 1 WHERE place_id = ?`, checkin.PlaceID)
}

func (s *Store) Checkins(lat, lon float64, radius int, limit int) ([]model.Checkin, error) {
	rows, err := s.db.Query(`SELECT id, uid, place_id, date, text, latitude, longitude FROM checkins
		ORDER BY date DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Checkin
	for rows.Next() {
		var c model.Checkin
		if err := rows.Scan(&c.ID, &c.UID, &c.PlaceID, &c.Date, &c.Text, &c.Latitude, &c.Longitude); err != nil {
			return nil, err
		}
		if radius <= 0 || distance(lat, lon, c.Latitude, c.Longitude) <= float64(radius)*1000 {
			out = append(out, c)
		}
	}
	return out, rows.Err()
}

func (s *Store) AddGroup(group model.Group) error {
	_, err := s.db.Exec(`INSERT INTO groups(gid, name, is_closed, is_admin) VALUES(?,?,?,?)
		ON CONFLICT(gid) DO UPDATE SET name = excluded.name, is_closed = excluded.is_closed,
			is_admin = excluded.is_admin`,
		group.GID, group.Name, boolToInt(group.IsClosed), boolToInt(group.IsAdmin))
	return err
}

func (s *Store) Groups(ids []int) ([]model.Group, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT gid, name, is_closed, is_admin FROM groups WHERE gid IN (`+
		placeholders(len(ids))+`)`, intsToAny(ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Group
	for rows.Next() {
		var g model.Group
		var closed, admin int
		if err := rows.Scan(&g.GID, &g.Name, &closed, &admin); err != nil {
			return nil, err
		}
		g.IsClosed = closed == 1
		g.IsAdmin = admin == 1
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) AddPoll(poll model.Poll) error {
	if _, err := s.db.Exec(`INSERT INTO polls(owner_id, poll_id, question) VALUES(?,?,?)
		ON CONFLICT(owner_id, poll_id) DO UPDATE SET question = excluded.question`,
		poll.OwnerID, poll.PollID, poll.Question); err != nil {
		return err
	}
	for _, answer := range poll.Answers {
		if _, err := s.db.Exec(`INSERT INTO poll_answers(owner_id, poll_id, answer_id, text, votes)
			VALUES(?,?,?,?,?)
			ON CONFLICT(owner_id, poll_id, answer_id) DO UPDATE SET text = excluded.text`,
			poll.OwnerID, poll.PollID, answer.ID, answer.Text, answer.Votes); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) NextPollID() (int, error) {
	return s.nextID("poll")
}

func (s *Store) Poll(ownerID, pollID int) (model.Poll, error) {
	var poll model.Poll
	err := s.db.QueryRow(`SELECT owner_id, poll_id, question FROM polls WHERE owner_id = ? AND poll_id = ?`,
		ownerID, pollID).Scan(&poll.OwnerID, &poll.PollID, &poll.Question)
	if err == sql.ErrNoRows {
		return model.Poll{}, ErrNotFound
	}
	if err != nil {
		return model.Poll{}, err
	}
	rows, err := s.db.Query(`SELECT answer_id, text, votes FROM poll_answers
		WHERE owner_id = ? AND poll_id = ? ORDER BY answer_id`, ownerID, pollID)
	if err != nil {
		return model.Poll{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var answer model.PollAnswer
		if err := rows.Scan(&answer.ID, &answer.Text, &answer.Votes); err != nil {
			return model.Poll{}, err
		}
		poll.Answers = append(poll.Answers, answer)
	}
	return poll, rows.Err()
}

func (s *Store) PollVote(ownerID, pollID, uid, answerID int) error {
	if _, err := s.db.Exec(`INSERT INTO poll_votes(owner_id, poll_id, uid, answer_id) VALUES(?,?,?,?)
		ON CONFLICT(owner_id, poll_id, uid) DO UPDATE SET answer_id = excluded.answer_id`,
		ownerID, pollID, uid, answerID); err != nil {
		return err
	}
	return s.exec(`UPDATE poll_answers SET votes = (
			SELECT COUNT(*) FROM poll_votes WHERE owner_id = ? AND poll_id = ? AND answer_id = poll_answers.answer_id
		) WHERE owner_id = ? AND poll_id = ?`, ownerID, pollID, ownerID, pollID)
}

func (s *Store) PollAnswerID(ownerID, pollID, uid int) int {
	var answerID int
	if err := s.db.QueryRow(`SELECT answer_id FROM poll_votes WHERE owner_id = ? AND poll_id = ? AND uid = ?`,
		ownerID, pollID, uid).Scan(&answerID); err != nil {
		return 0
	}
	return answerID
}

func (s *Store) PollTotalVotes(ownerID, pollID int) int {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM poll_votes WHERE owner_id = ? AND poll_id = ?`,
		ownerID, pollID).Scan(&total); err != nil {
		return 0
	}
	return total
}

func collectPlaces(rows *sql.Rows) ([]model.Place, error) {
	var out []model.Place
	for rows.Next() {
		var p model.Place
		if err := rows.Scan(&p.PlaceID, &p.Title, &p.Address, &p.Latitude, &p.Longitude, &p.Type,
			&p.Checkins, &p.CityID, &p.CountryID); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
