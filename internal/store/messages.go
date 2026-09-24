package store

import (
	"database/sql"

	"vkemu/internal/model"
)

const messageColumns = `mid, from_id, to_id, chat_id, date, body, read_state, attach_json`

func (s *Store) AddMessage(msg model.Message) (int, error) {
	mid, err := s.nextID("message")
	if err != nil {
		return 0, err
	}
	msg.MID = mid
	if msg.Date == 0 {
		msg.Date = s.Now()
	}
	_, err = s.db.Exec(`INSERT INTO messages(`+messageColumns+`) VALUES(?,?,?,?,?,?,?,?)`,
		msg.MID, msg.FromID, msg.ToID, msg.ChatID, msg.Date, msg.Body, boolToInt(msg.ReadState), msg.AttachJSON)
	return msg.MID, err
}

func (s *Store) Message(mid int) (model.Message, error) {
	row := s.db.QueryRow(`SELECT `+messageColumns+` FROM messages WHERE mid = ?`, mid)
	return scanMessage(row)
}

func (s *Store) Messages(peer int, offset, count int) ([]model.Message, error) {
	rows, err := s.db.Query(`SELECT `+messageColumns+` FROM messages
		WHERE from_id = ? OR to_id = ?
		ORDER BY mid DESC LIMIT ? OFFSET ?`, peer, peer, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func (s *Store) Conversation(uid, peer, offset, count int) ([]model.Message, error) {
	rows, err := s.db.Query(`SELECT `+messageColumns+` FROM messages
		WHERE (from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?)
		ORDER BY mid DESC LIMIT ? OFFSET ?`, uid, peer, peer, uid, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func (s *Store) ConversationCount(uid, peer int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM messages
		WHERE (from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?)`, uid, peer, peer, uid).
		Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) MessagesSince(uid int, since int64, limit int) ([]model.Message, error) {
	rows, err := s.db.Query(`SELECT `+messageColumns+` FROM messages
		WHERE date >= ? AND (from_id = ? OR to_id = ?) ORDER BY mid DESC LIMIT ?`,
		since, uid, uid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func (s *Store) MessagesByID(ids []int) ([]model.Message, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT `+messageColumns+` FROM messages WHERE mid IN (`+placeholders(len(ids))+`)`,
		intsToAny(ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func (s *Store) SearchMessages(uid int, query string, offset, count int) ([]model.Message, error) {
	pattern := "%" + query + "%"
	rows, err := s.db.Query(`SELECT `+messageColumns+` FROM messages
		WHERE body LIKE ? AND (from_id = ? OR to_id = ?) ORDER BY mid DESC LIMIT ? OFFSET ?`,
		pattern, uid, uid, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func (s *Store) DeleteMessage(mid int) error {
	return s.exec(`DELETE FROM messages WHERE mid = ?`, mid)
}

func (s *Store) MarkMessagesRead(ids []int, uid int) error {
	if len(ids) == 0 {
		return nil
	}
	args := intsToAny(ids)
	args = append(args, uid)
	return s.exec(`UPDATE messages SET read_state = 1 WHERE mid IN (`+placeholders(len(ids))+`) AND to_id = ?`,
		args...)
}

func (s *Store) UnreadCount(uid int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE to_id = ? AND read_state = 0`, uid).
		Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) Dialogs(uid int, offset, count int) ([]model.Message, error) {
	rows, err := s.db.Query(`SELECT `+messageColumns+` FROM messages
		WHERE mid IN (
			SELECT MAX(mid) FROM messages WHERE
				(chat_id = 0 AND (from_id = ? OR to_id = ?)) OR
				(chat_id != 0 AND chat_id IN (SELECT DISTINCT chat_id FROM messages WHERE chat_id != 0 AND from_id = ?))
			GROUP BY CASE WHEN chat_id != 0 THEN 2000000000 + chat_id
				ELSE CASE WHEN from_id = ? THEN to_id ELSE from_id END END
		)
		ORDER BY date DESC LIMIT ? OFFSET ?`, uid, uid, uid, uid, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func (s *Store) DialogCount(uid int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM (
		SELECT MAX(mid) FROM messages WHERE
			(chat_id = 0 AND (from_id = ? OR to_id = ?)) OR
			(chat_id != 0 AND chat_id IN (SELECT DISTINCT chat_id FROM messages WHERE chat_id != 0 AND from_id = ?))
		GROUP BY CASE WHEN chat_id != 0 THEN 2000000000 + chat_id
			ELSE CASE WHEN from_id = ? THEN to_id ELSE from_id END END)`, uid, uid, uid, uid).
		Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) ChatMessages(chatID, offset, count int) ([]model.Message, error) {
	rows, err := s.db.Query(`SELECT `+messageColumns+` FROM messages WHERE chat_id = ?
		ORDER BY mid DESC LIMIT ? OFFSET ?`, chatID, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func (s *Store) ChatMembers(chatID int) ([]int, error) {
	rows, err := s.db.Query(`SELECT DISTINCT from_id FROM messages WHERE chat_id = ? ORDER BY from_id`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectInts(rows)
}

func (s *Store) ChatTitle(chatID int) string {
	var title string
	if err := s.db.QueryRow(`SELECT body FROM messages WHERE chat_id = ? ORDER BY mid LIMIT 1`, chatID).
		Scan(&title); err != nil {
		return "Chat"
	}
	return title
}

func scanMessage(row interface{ Scan(...any) error }) (model.Message, error) {
	var m model.Message
	var readState int
	err := row.Scan(&m.MID, &m.FromID, &m.ToID, &m.ChatID, &m.Date, &m.Body, &readState, &m.AttachJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Message{}, ErrNotFound
		}
		return model.Message{}, err
	}
	m.ReadState = readState == 1
	return m, nil
}

func collectMessages(rows *sql.Rows) ([]model.Message, error) {
	var out []model.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
