package api

import (
	"vkemu/internal/model"
)

func (s *Server) placesSearch(c *call) (any, error) {
	lat := c.p.Float("latitude", 0)
	lon := c.p.Float("longitude", 0)
	radius := c.p.Int("radius", 1)

	places, err := s.db.SearchPlaces(lat, lon, radius)
	if err != nil {
		return nil, err
	}
	if len(places) == 0 {
		places, err = s.db.SearchPlaces(lat, lon, 3)
		if err != nil {
			return nil, err
		}
	}

	out := make([]any, 0, len(places)+1)
	out = append(out, len(places))
	for _, place := range places {
		entry := c.formatPlace(place)
		entry["distance"] = distanceMeters(lat, lon, place.Latitude, place.Longitude)
		out = append(out, entry)
	}
	return out, nil
}

func (s *Server) placesGetByID(c *call) (any, error) {
	ids := c.p.IntList("places")
	places, err := s.db.Places(ids)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(places))
	for _, place := range places {
		out = append(out, c.formatPlace(place))
	}
	return out, nil
}

func (s *Server) placesAdd(c *call) (any, error) {
	placeID, err := s.db.NextPlaceID()
	if err != nil {
		return nil, err
	}
	place := model.Place{
		PlaceID:   placeID,
		Title:     c.p.String("title", "Новое место"),
		Address:   c.p.String("address", ""),
		Latitude:  c.p.Float("latitude", 0),
		Longitude: c.p.Float("longitude", 0),
		Type:      c.p.Int("type", 1),
		CityID:    c.p.Int("city_id", 1),
		CountryID: c.p.Int("country_id", 1),
	}
	if err := s.db.AddPlace(place); err != nil {
		return nil, err
	}
	return map[string]any{"pid": placeID}, nil
}

func (s *Server) placesCheckin(c *call) (any, error) {
	placeID := c.p.Int("place_id", 0)
	place, err := s.db.Place(placeID)
	if err != nil {
		return nil, errNotFound
	}
	text := c.p.String("text", "")
	checkin := model.Checkin{
		UID:       c.viewer(),
		PlaceID:   placeID,
		Date:      s.db.Now(),
		Text:      text,
		Latitude:  place.Latitude,
		Longitude: place.Longitude,
	}
	if err := s.db.AddCheckin(checkin); err != nil {
		return nil, err
	}

	postID, err := s.db.NextPostID(c.viewer())
	if err != nil {
		return nil, err
	}
	post := model.WallPost{
		OwnerID:    c.viewer(),
		PostID:     postID,
		FromID:     c.viewer(),
		Date:       s.db.Now(),
		Text:       text,
		GeoJSON:    `{"place":{"place_id":` + itoa(placeID) + `,"title":"` + place.Title + `"}}`,
		AttachType: "geo",
	}
	if err := s.db.CreatePost(post); err != nil {
		return nil, err
	}

	out := make([]any, 0, 1)
	out = append(out, map[string]any{"cid": postID})
	return out, nil
}

func (s *Server) placesGetCheckins(c *call) (any, error) {
	lat := c.p.Float("latitude", 0)
	lon := c.p.Float("longitude", 0)
	radius := c.p.Int("radius", 0)
	limit := c.p.Int("count", 100)

	checkins, err := s.db.Checkins(lat, lon, radius, limit)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(checkins)+1)
	out = append(out, len(checkins))
	for _, checkin := range checkins {
		place, err := s.db.Place(checkin.PlaceID)
		if err != nil {
			continue
		}
		out = append(out, map[string]any{
			"id":        checkin.ID,
			"uid":       checkin.UID,
			"place_id":  checkin.PlaceID,
			"date":      int(checkin.Date),
			"text":      checkin.Text,
			"latitude":  checkin.Latitude,
			"longitude": checkin.Longitude,
			"place":     c.formatPlace(place),
		})
	}
	return out, nil
}

func (s *Server) placesGetCityByID(c *call) (any, error) {
	ids := c.p.IntList("cids")
	cities, err := s.db.Cities(ids)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(cities))
	for _, city := range cities {
		out = append(out, c.formatCity(city))
	}
	return out, nil
}

func (s *Server) placesGetCountryByID(c *call) (any, error) {
	ids := c.p.IntList("cids")
	countries, err := s.db.Countries(ids)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(countries))
	for _, country := range countries {
		out = append(out, c.formatCountry(country))
	}
	return out, nil
}

func distanceMeters(lat1, lon1, lat2, lon2 float64) int {
	return distance(lat1, lon1, lat2, lon2)
}
