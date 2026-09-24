package api

import (
	"math"
	"strconv"
)

const earthRadius = 6371000.0

func distance(lat1, lon1, lat2, lon2 float64) int {
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	dPhi := (lat2 - lat1) * math.Pi / 180
	dLambda := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) + math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	return int(math.Round(earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))))
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
