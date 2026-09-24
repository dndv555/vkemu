package store

import (
	"database/sql"

	"vkemu/internal/model"
)

func (s *Store) AddAudio(item model.Audio) error {
	_, err := s.db.Exec(`INSERT INTO audio(owner_id, aid, album_id, artist, title, duration, url)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(owner_id, aid) DO UPDATE SET album_id = excluded.album_id, artist = excluded.artist,
			title = excluded.title, duration = excluded.duration, url = excluded.url`,
		item.OwnerID, item.AID, item.AlbumID, item.Artist, item.Title, item.Duration, item.URL)
	return err
}

func (s *Store) NextAudioID() (int, error) {
	return s.nextID("audio")
}

func (s *Store) Audio(ownerID int, offset, count int, albumID int) ([]model.Audio, error) {
	query := `SELECT owner_id, aid, album_id, artist, title, duration, url FROM audio WHERE owner_id = ?`
	args := []any{ownerID}
	if albumID > 0 {
		query += ` AND album_id = ?`
		args = append(args, albumID)
	}
	query += ` ORDER BY aid DESC LIMIT ? OFFSET ?`
	args = append(args, count, offset)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectAudio(rows)
}

func (s *Store) AudioByID(pairs [][2]int) ([]model.Audio, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	var out []model.Audio
	for _, pair := range pairs {
		var item model.Audio
		err := s.db.QueryRow(`SELECT owner_id, aid, album_id, artist, title, duration, url
			FROM audio WHERE owner_id = ? AND aid = ?`, pair[0], pair[1]).
			Scan(&item.OwnerID, &item.AID, &item.AlbumID, &item.Artist, &item.Title, &item.Duration, &item.URL)
		if err == nil {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *Store) SearchAudio(query string, offset, count int) ([]model.Audio, error) {
	pattern := "%" + query + "%"
	rows, err := s.db.Query(`SELECT owner_id, aid, album_id, artist, title, duration, url FROM audio
		WHERE artist LIKE ? OR title LIKE ? ORDER BY aid LIMIT ? OFFSET ?`, pattern, pattern, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectAudio(rows)
}

func (s *Store) DeleteAudio(ownerID, aid int) error {
	return s.exec(`DELETE FROM audio WHERE owner_id = ? AND aid = ?`, ownerID, aid)
}

func (s *Store) AudioAlbums(ownerID int) ([]model.AudioAlbum, error) {
	rows, err := s.db.Query(`SELECT owner_id, album_id, title FROM audio_albums WHERE owner_id = ? ORDER BY album_id`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AudioAlbum
	for rows.Next() {
		var a model.AudioAlbum
		if err := rows.Scan(&a.OwnerID, &a.AlbumID, &a.Title); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AddAudioAlbum(album model.AudioAlbum) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO audio_albums(owner_id, album_id, title) VALUES(?,?,?)`,
		album.OwnerID, album.AlbumID, album.Title)
	return err
}

func (s *Store) AddAlbum(album model.Album) error {
	_, err := s.db.Exec(`INSERT INTO albums(owner_id, aid, title, description, privacy, thumb_id, created)
		VALUES(?,?,?,?,?,?,?)`, album.OwnerID, album.AID, album.Title, album.Description,
		album.Privacy, album.ThumbID, album.Created)
	return err
}

func (s *Store) NextAlbumID() (int, error) {
	return s.nextID("album")
}

func (s *Store) Albums(ownerID int) ([]model.Album, error) {
	rows, err := s.db.Query(`SELECT owner_id, aid, title, description, privacy, thumb_id, created
		FROM albums WHERE owner_id = ? ORDER BY aid`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Album
	for rows.Next() {
		var a model.Album
		if err := rows.Scan(&a.OwnerID, &a.AID, &a.Title, &a.Description, &a.Privacy, &a.ThumbID, &a.Created); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) Album(ownerID, aid int) (model.Album, error) {
	var a model.Album
	err := s.db.QueryRow(`SELECT owner_id, aid, title, description, privacy, thumb_id, created
		FROM albums WHERE owner_id = ? AND aid = ?`, ownerID, aid).
		Scan(&a.OwnerID, &a.AID, &a.Title, &a.Description, &a.Privacy, &a.ThumbID, &a.Created)
	if err == sql.ErrNoRows {
		return model.Album{}, ErrNotFound
	}
	if err != nil {
		return model.Album{}, err
	}
	return a, nil
}

func (s *Store) UpdateAlbum(ownerID, aid int, title, description string, privacy int) error {
	return s.exec(`UPDATE albums SET title = ?, description = ?, privacy = ? WHERE owner_id = ? AND aid = ?`,
		title, description, privacy, ownerID, aid)
}

func (s *Store) DeleteAlbum(ownerID, aid int) error {
	if err := s.exec(`DELETE FROM albums WHERE owner_id = ? AND aid = ?`, ownerID, aid); err != nil {
		return err
	}
	return s.exec(`DELETE FROM photos WHERE owner_id = ? AND aid = ?`, ownerID, aid)
}

func (s *Store) AddPhoto(photo model.Photo) error {
	_, err := s.db.Exec(`INSERT INTO photos(owner_id, pid, aid, date, text, src, src_big, likes, comments)
		VALUES(?,?,?,?,?,?,?,?,?)`, photo.OwnerID, photo.PID, photo.AID, photo.Date, photo.Text,
		photo.Src, photo.SrcBig, photo.Likes, photo.Comments)
	return err
}

func (s *Store) NextPhotoID() (int, error) {
	return s.nextID("photo")
}

func (s *Store) Photos(ownerID, aid, offset, count int) ([]model.Photo, error) {
	query := `SELECT owner_id, pid, aid, date, text, src, src_big, likes, comments FROM photos WHERE owner_id = ?`
	args := []any{ownerID}
	if aid != 0 {
		query += ` AND aid = ?`
		args = append(args, aid)
	}
	query += ` ORDER BY pid DESC LIMIT ? OFFSET ?`
	args = append(args, count, offset)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPhotos(rows)
}

func (s *Store) PhotoByID(pairs [][2]int) ([]model.Photo, error) {
	var out []model.Photo
	for _, pair := range pairs {
		var p model.Photo
		err := s.db.QueryRow(`SELECT owner_id, pid, aid, date, text, src, src_big, likes, comments
			FROM photos WHERE owner_id = ? AND pid = ?`, pair[0], pair[1]).
			Scan(&p.OwnerID, &p.PID, &p.AID, &p.Date, &p.Text, &p.Src, &p.SrcBig, &p.Likes, &p.Comments)
		if err == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Store) PhotoCount(ownerID, aid int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM photos WHERE owner_id = ? AND aid = ?`, ownerID, aid).
		Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) PhotoCountAll(ownerID int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM photos WHERE owner_id = ?`, ownerID).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) DeletePhoto(ownerID, pid int) error {
	return s.exec(`DELETE FROM photos WHERE owner_id = ? AND pid = ?`, ownerID, pid)
}

func (s *Store) SetPhotoText(ownerID, pid int, text string) error {
	return s.exec(`UPDATE photos SET text = ? WHERE owner_id = ? AND pid = ?`, text, ownerID, pid)
}

func (s *Store) AddVideo(video model.Video) error {
	_, err := s.db.Exec(`INSERT INTO videos(owner_id, vid, title, description, duration, image, url, date)
		VALUES(?,?,?,?,?,?,?,?)`, video.OwnerID, video.VID, video.Title, video.Description,
		video.Duration, video.Image, video.URL, video.Date)
	return err
}

func (s *Store) NextVideoID() (int, error) {
	return s.nextID("video")
}

func (s *Store) Video(ownerID, vid int) (model.Video, error) {
	var v model.Video
	err := s.db.QueryRow(`SELECT owner_id, vid, title, description, duration, image, url, date
		FROM videos WHERE owner_id = ? AND vid = ?`, ownerID, vid).
		Scan(&v.OwnerID, &v.VID, &v.Title, &v.Description, &v.Duration, &v.Image, &v.URL, &v.Date)
	if err == sql.ErrNoRows {
		return model.Video{}, ErrNotFound
	}
	if err != nil {
		return model.Video{}, err
	}
	return v, nil
}

func (s *Store) Videos(ownerID, offset, count int) ([]model.Video, error) {
	rows, err := s.db.Query(`SELECT owner_id, vid, title, description, duration, image, url, date
		FROM videos WHERE owner_id = ? ORDER BY vid DESC LIMIT ? OFFSET ?`, ownerID, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.OwnerID, &v.VID, &v.Title, &v.Description, &v.Duration,
			&v.Image, &v.URL, &v.Date); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) UpdateVideo(video model.Video) error {
	return s.exec(`UPDATE videos SET title = ?, description = ?, duration = ?, image = ?, url = ?
		WHERE owner_id = ? AND vid = ?`, video.Title, video.Description, video.Duration,
		video.Image, video.URL, video.OwnerID, video.VID)
}

func (s *Store) AddNote(note model.Note) error {
	_, err := s.db.Exec(`INSERT INTO notes(owner_id, nid, title, text, date, comments) VALUES(?,?,?,?,?,?)`,
		note.OwnerID, note.NID, note.Title, note.Text, note.Date, note.Comments)
	return err
}

func (s *Store) NextNoteID() (int, error) {
	return s.nextID("note")
}

func (s *Store) Notes(ownerID, offset, count int) ([]model.Note, error) {
	rows, err := s.db.Query(`SELECT owner_id, nid, title, text, date, comments FROM notes
		WHERE owner_id = ? ORDER BY nid DESC LIMIT ? OFFSET ?`, ownerID, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Note
	for rows.Next() {
		var n model.Note
		if err := rows.Scan(&n.OwnerID, &n.NID, &n.Title, &n.Text, &n.Date, &n.Comments); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) NoteCount(ownerID int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM notes WHERE owner_id = ?`, ownerID).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) DeleteNote(ownerID, nid int) error {
	return s.exec(`DELETE FROM notes WHERE owner_id = ? AND nid = ?`, ownerID, nid)
}

func (s *Store) Note(ownerID, nid int) (model.Note, error) {
	var n model.Note
	err := s.db.QueryRow(`SELECT owner_id, nid, title, text, date, comments FROM notes
		WHERE owner_id = ? AND nid = ?`, ownerID, nid).
		Scan(&n.OwnerID, &n.NID, &n.Title, &n.Text, &n.Date, &n.Comments)
	if err == sql.ErrNoRows {
		return model.Note{}, ErrNotFound
	}
	if err != nil {
		return model.Note{}, err
	}
	return n, nil
}

func (s *Store) AddDoc(doc model.Doc) error {
	_, err := s.db.Exec(`INSERT INTO docs(owner_id, did, title, ext, size, url, date) VALUES(?,?,?,?,?,?,?)`,
		doc.OwnerID, doc.DID, doc.Title, doc.Ext, doc.Size, doc.URL, doc.Date)
	return err
}

func (s *Store) NextDocID() (int, error) {
	return s.nextID("doc")
}

func (s *Store) Docs(ownerID, offset, count int) ([]model.Doc, error) {
	rows, err := s.db.Query(`SELECT owner_id, did, title, ext, size, url, date FROM docs
		WHERE owner_id = ? ORDER BY did DESC LIMIT ? OFFSET ?`, ownerID, count, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Doc
	for rows.Next() {
		var d model.Doc
		if err := rows.Scan(&d.OwnerID, &d.DID, &d.Title, &d.Ext, &d.Size, &d.URL, &d.Date); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) DocCount(ownerID int) int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM docs WHERE owner_id = ?`, ownerID).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) DeleteDoc(ownerID, did int) error {
	return s.exec(`DELETE FROM docs WHERE owner_id = ? AND did = ?`, ownerID, did)
}

func (s *Store) Doc(ownerID, did int) (model.Doc, error) {
	var d model.Doc
	err := s.db.QueryRow(`SELECT owner_id, did, title, ext, size, url, date FROM docs
		WHERE owner_id = ? AND did = ?`, ownerID, did).
		Scan(&d.OwnerID, &d.DID, &d.Title, &d.Ext, &d.Size, &d.URL, &d.Date)
	if err == sql.ErrNoRows {
		return model.Doc{}, ErrNotFound
	}
	if err != nil {
		return model.Doc{}, err
	}
	return d, nil
}

func collectAudio(rows *sql.Rows) ([]model.Audio, error) {
	var out []model.Audio
	for rows.Next() {
		var a model.Audio
		if err := rows.Scan(&a.OwnerID, &a.AID, &a.AlbumID, &a.Artist, &a.Title, &a.Duration, &a.URL); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func collectPhotos(rows *sql.Rows) ([]model.Photo, error) {
	var out []model.Photo
	for rows.Next() {
		var p model.Photo
		if err := rows.Scan(&p.OwnerID, &p.PID, &p.AID, &p.Date, &p.Text, &p.Src, &p.SrcBig,
			&p.Likes, &p.Comments); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
