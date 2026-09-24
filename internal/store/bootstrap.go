package store

import "vkemu/internal/model"

var bootstrapCities = []model.City{
	{CID: 1, Name: "Москва"},
	{CID: 2, Name: "Санкт-Петербург"},
	{CID: 3, Name: "Новосибирск"},
	{CID: 4, Name: "Екатеринбург"},
	{CID: 5, Name: "Казань"},
}

var bootstrapCountries = []model.Country{
	{CID: 1, Name: "Россия"},
	{CID: 2, Name: "Беларусь"},
	{CID: 3, Name: "Казахстан"},
}

func (s *Store) Bootstrap() error {
	for _, city := range bootstrapCities {
		if err := s.AddCity(city); err != nil {
			return err
		}
	}
	for _, country := range bootstrapCountries {
		if err := s.AddCountry(country); err != nil {
			return err
		}
	}
	return nil
}
