package store

import (
	"database/sql"
	"errors"

	"vkemu/internal/model"
)

const postColumns = `owner_id, post_id, from_id, date, text, likes, comments,
	copy_owner_id, copy_post_id, attach_type, attach_json, geo_json, deleted`

func (s *Store) NextPostID(ownerID int) (int, error) {
	return s.nextID("wall_post")
}

func (s *Store) CreatePost(post model.WallPost) error {
	_, err := s.db.Exec(`INSERT INTO wall_posts(`+postColumns+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		post.OwnerID, post.PostID, post.FromID, post.Date, post.Text, post.Likes, post.Comments,
		post.CopyOwnerID, post.CopyPostID, post.AttachType, post.AttachJSON, post.GeoJSON, boolToInt(post.Deleted))
	return err
}

func (s *Store) Post(ownerID, postID int) (model.WallPost, error) {
	row := s.db.QueryRow(`SELECT `+postColumns+` FROM wall_posts WHERE owner_id = ? AND post_id = ?`,
		ownerID, postID)
	return scanPost(row)
}

func (s *Store) Posts(ownerID int, offset, count int) ([]model.WallPost, error) {
	rows, err := s.db.Query(`SELECT `+postColumns+` FROM wall_posts
		WHERE owner_id = ? AND deleted = 0 ORDER BY post_id DESC LIMIT ? OFFSET ?`,
		ownerID, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPosts(rows)
}

func (s *Store) PostsByID(ids []int) ([]model.WallPost, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT `+postColumns+` FROM wall_posts
		WHERE post_id IN (`+placeholders(len(ids))+`) AND deleted = 0 ORDER BY post_id DESC`,
		intsToAny(ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPosts(rows)
}

func (s *Store) Feed(uid int, offset, count int) ([]model.WallPost, error) {
	return s.FeedRange(uid, 0, 0, offset, count)
}

func (s *Store) FeedRange(uid int, endTime, startTime, offset, count int) ([]model.WallPost, error) {
	query := `SELECT ` + postColumns + ` FROM wall_posts
		WHERE deleted = 0 AND (owner_id = ? OR owner_id IN (SELECT fid FROM friends WHERE uid = ?))`
	args := []any{uid, uid}
	if endTime > 0 {
		query += ` AND date <= ?`
		args = append(args, endTime)
	}
	if startTime > 0 {
		query += ` AND date >= ?`
		args = append(args, startTime)
	}
	query += ` ORDER BY date DESC, post_id DESC LIMIT ? OFFSET ?`
	args = append(args, count, offset)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPosts(rows)
}

func (s *Store) PostCount(ownerID int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM wall_posts WHERE owner_id = ? AND deleted = 0`, ownerID).
		Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) DeletePost(ownerID, postID int) error {
	return s.exec(`UPDATE wall_posts SET deleted = 1 WHERE owner_id = ? AND post_id = ?`, ownerID, postID)
}

func (s *Store) UpdatePost(post model.WallPost) error {
	return s.exec(`UPDATE wall_posts SET text = ?, likes = ?, comments = ?, copy_owner_id = ?,
		copy_post_id = ?, attach_type = ?, attach_json = ?, geo_json = ? WHERE owner_id = ? AND post_id = ?`,
		post.Text, post.Likes, post.Comments, post.CopyOwnerID, post.CopyPostID, post.AttachType,
		post.AttachJSON, post.GeoJSON, post.OwnerID, post.PostID)
}

func (s *Store) SetPostCounters(ownerID, postID, likes, comments int) error {
	return s.exec(`UPDATE wall_posts SET likes = ?, comments = ? WHERE owner_id = ? AND post_id = ?`,
		likes, comments, ownerID, postID)
}

func (s *Store) Comments(cType string, ownerID, itemID, offset, count int) ([]model.Comment, error) {
	rows, err := s.db.Query(`SELECT ctype, owner_id, item_id, cid, from_id, date, text, reply_to_uid
		FROM comments WHERE ctype = ? AND owner_id = ? AND item_id = ?
		ORDER BY cid LIMIT ? OFFSET ?`, cType, ownerID, itemID, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectComments(rows)
}

func (s *Store) CommentCount(cType string, ownerID, itemID int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM comments WHERE ctype = ? AND owner_id = ? AND item_id = ?`,
		cType, ownerID, itemID).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) AddComment(comment model.Comment) error {
	cid, err := s.nextID("comment")
	if err != nil {
		return err
	}
	comment.CID = cid
	_, err = s.db.Exec(`INSERT INTO comments(ctype, owner_id, item_id, cid, from_id, date, text, reply_to_uid)
		VALUES(?,?,?,?,?,?,?,?)`, comment.CType, comment.OwnerID, comment.ItemID, comment.CID,
		comment.FromID, comment.Date, comment.Text, comment.ReplyToUID)
	return err
}

func (s *Store) Comment(cType string, ownerID, itemID, cid int) (model.Comment, error) {
	var c model.Comment
	err := s.db.QueryRow(`SELECT ctype, owner_id, item_id, cid, from_id, date, text, reply_to_uid
		FROM comments WHERE ctype = ? AND owner_id = ? AND item_id = ? AND cid = ?`,
		cType, ownerID, itemID, cid).
		Scan(&c.CType, &c.OwnerID, &c.ItemID, &c.CID, &c.FromID, &c.Date, &c.Text, &c.ReplyToUID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Comment{}, ErrNotFound
	}
	if err != nil {
		return model.Comment{}, err
	}
	return c, nil
}

func (s *Store) DeleteComment(cType string, ownerID, itemID, cid int) error {
	return s.exec(`DELETE FROM comments WHERE ctype = ? AND owner_id = ? AND item_id = ? AND cid = ?`,
		cType, ownerID, itemID, cid)
}

func (s *Store) Like(cType string, ownerID, itemID, uid int) (bool, int) {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO likes(ctype, owner_id, item_id, uid) VALUES(?,?,?,?)`,
		cType, ownerID, itemID, uid)
	if err == nil {
		if affected, _ := res.RowsAffected(); affected > 0 {
			s.exec(`UPDATE wall_posts SET likes = (SELECT COUNT(*) FROM likes
				WHERE ctype = 'post' AND owner_id = ? AND item_id = ?) WHERE owner_id = ? AND post_id = ?`,
				ownerID, itemID, ownerID, itemID)
		}
	}
	return s.IsLiked(cType, ownerID, itemID, uid), s.LikeCount(cType, ownerID, itemID)
}

func (s *Store) Unlike(cType string, ownerID, itemID, uid int) (bool, int) {
	if err := s.exec(`DELETE FROM likes WHERE ctype = ? AND owner_id = ? AND item_id = ? AND uid = ?`,
		cType, ownerID, itemID, uid); err != nil {
		return false, s.LikeCount(cType, ownerID, itemID)
	}
	s.exec(`UPDATE wall_posts SET likes = (SELECT COUNT(*) FROM likes
		WHERE ctype = 'post' AND owner_id = ? AND item_id = ?) WHERE owner_id = ? AND post_id = ?`,
		ownerID, itemID, ownerID, itemID)
	return false, s.LikeCount(cType, ownerID, itemID)
}

func (s *Store) LikeCount(cType string, ownerID, itemID int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM likes WHERE ctype = ? AND owner_id = ? AND item_id = ?`,
		cType, ownerID, itemID).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) IsLiked(cType string, ownerID, itemID, uid int) bool {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM likes
		WHERE ctype = ? AND owner_id = ? AND item_id = ? AND uid = ?`,
		cType, ownerID, itemID, uid).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func scanPost(row interface{ Scan(...any) error }) (model.WallPost, error) {
	var p model.WallPost
	var deleted int
	err := row.Scan(&p.OwnerID, &p.PostID, &p.FromID, &p.Date, &p.Text, &p.Likes, &p.Comments,
		&p.CopyOwnerID, &p.CopyPostID, &p.AttachType, &p.AttachJSON, &p.GeoJSON, &deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return model.WallPost{}, ErrNotFound
	}
	if err != nil {
		return model.WallPost{}, err
	}
	p.Deleted = deleted == 1
	return p, nil
}

func collectPosts(rows *sql.Rows) ([]model.WallPost, error) {
	var out []model.WallPost
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func collectComments(rows *sql.Rows) ([]model.Comment, error) {
	var out []model.Comment
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.CType, &c.OwnerID, &c.ItemID, &c.CID, &c.FromID, &c.Date,
			&c.Text, &c.ReplyToUID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
