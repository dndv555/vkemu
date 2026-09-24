package store

import (
	"math"
	"time"

	"vkemu/internal/model"
)

const earthRadius = 6371000.0

func distance(lat1, lon1, lat2, lon2 float64) float64 {
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	dPhi := (lat2 - lat1) * math.Pi / 180
	dLambda := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) + math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func (s *Store) SaveUpload(upload model.Upload) error {
	if upload.Created == 0 {
		upload.Created = time.Now().Unix()
	}
	_, err := s.db.Exec(`INSERT INTO uploads(hash, kind, owner_id, path, size, title, created)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(hash) DO UPDATE SET kind = excluded.kind, owner_id = excluded.owner_id,
			path = excluded.path, size = excluded.size, title = excluded.title`,
		upload.Hash, upload.Kind, upload.OwnerID, upload.Path, upload.Size, upload.Title, upload.Created)
	return err
}

func (s *Store) Upload(hash string) (model.Upload, bool) {
	var u model.Upload
	err := s.db.QueryRow(`SELECT hash, kind, owner_id, path, size, title, created FROM uploads WHERE hash = ?`, hash).
		Scan(&u.Hash, &u.Kind, &u.OwnerID, &u.Path, &u.Size, &u.Title, &u.Created)
	if err != nil {
		return model.Upload{}, false
	}
	return u, true
}
