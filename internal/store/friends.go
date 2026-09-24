package store

import (
	"database/sql"
	"errors"

	"vkemu/internal/model"
)

func (s *Store) SaveSession(session model.Session) error {
	_, err := s.db.Exec(`INSERT INTO sessions(sid, uid, secret, created) VALUES(?,?,?,?)
		ON CONFLICT(sid) DO UPDATE SET uid = excluded.uid, secret = excluded.secret`,
		session.SID, session.UID, session.Secret, session.Created)
	return err
}

func (s *Store) Session(sid string) (model.Session, error) {
	var session model.Session
	err := s.db.QueryRow(`SELECT sid, uid, secret, created FROM sessions WHERE sid = ?`, sid).
		Scan(&session.SID, &session.UID, &session.Secret, &session.Created)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Session{}, ErrNotFound
	}
	if err != nil {
		return model.Session{}, err
	}
	return session, nil
}

func (s *Store) DeleteSession(sid string) error {
	return s.exec(`DELETE FROM sessions WHERE sid = ?`, sid)
}

func (s *Store) UpsertFriend(uid, fid int) error {
	if uid == fid {
		return nil
	}
	_, err := s.db.Exec(`INSERT OR IGNORE INTO friends(uid, fid) VALUES(?,?)`, uid, fid)
	return err
}

func (s *Store) DeleteFriend(uid, fid int) error {
	if err := s.exec(`DELETE FROM friends WHERE (uid = ? AND fid = ?) OR (uid = ? AND fid = ?)`,
		uid, fid, fid, uid); err != nil {
		return err
	}
	return s.DeleteFriendRequest(fid, uid)
}

func (s *Store) FriendIDs(uid int) ([]int, error) {
	rows, err := s.db.Query(`SELECT fid FROM friends WHERE uid = ? ORDER BY fid`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectInts(rows)
}

func (s *Store) AreFriends(uid, fid int) bool {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM friends WHERE uid = ? AND fid = ?`, uid, fid).
		Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func (s *Store) MutualFriendIDs(uid, target int) ([]int, error) {
	rows, err := s.db.Query(`SELECT a.fid FROM friends a
		JOIN friends b ON b.fid = a.fid
		WHERE a.uid = ? AND b.uid = ?
		ORDER BY a.fid`, uid, target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectInts(rows)
}

func (s *Store) SaveFriendRequest(req model.FriendRequest) error {
	_, err := s.db.Exec(`INSERT INTO friend_requests(from_uid, to_uid, message, date) VALUES(?,?,?,?)
		ON CONFLICT(from_uid, to_uid) DO UPDATE SET message = excluded.message, date = excluded.date`,
		req.FromUID, req.ToUID, req.Message, req.Date)
	return err
}

func (s *Store) DeleteFriendRequest(fromUID, toUID int) error {
	return s.exec(`DELETE FROM friend_requests WHERE from_uid = ? AND to_uid = ?`, fromUID, toUID)
}

func (s *Store) FriendRequest(fromUID, toUID int) (model.FriendRequest, error) {
	var req model.FriendRequest
	err := s.db.QueryRow(`SELECT from_uid, to_uid, message, date FROM friend_requests
		WHERE from_uid = ? AND to_uid = ?`, fromUID, toUID).
		Scan(&req.FromUID, &req.ToUID, &req.Message, &req.Date)
	if errors.Is(err, sql.ErrNoRows) {
		return model.FriendRequest{}, ErrNotFound
	}
	if err != nil {
		return model.FriendRequest{}, err
	}
	return req, nil
}

func (s *Store) FriendRequests(uid int, offset, count int) ([]model.FriendRequest, error) {
	rows, err := s.db.Query(`SELECT from_uid, to_uid, message, date FROM friend_requests
		WHERE to_uid = ? ORDER BY date DESC LIMIT ? OFFSET ?`, uid, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.FriendRequest
	for rows.Next() {
		var req model.FriendRequest
		if err := rows.Scan(&req.FromUID, &req.ToUID, &req.Message, &req.Date); err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

func (s *Store) FriendRequestCount(uid int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM friend_requests WHERE to_uid = ?`, uid).
		Scan(&count); err != nil {
		return 0
	}
	return count
}
