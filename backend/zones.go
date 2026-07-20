package main

import "math"

const earthRadiusKm = 6371.0

// Zone is a circular region of interest. Circles are a deliberate
// simplification over arbitrary polygons — the containment check is one
// haversine distance calculation instead of ray-casting, and a circle is
// still expressive enough to represent a real advisory region.
//
// These three are real, publicly documented aviation advisory regions
// (published NOTAMs / airline route-avoidance zones), included here purely
// as illustrative geospatial data — not a political statement about any
// conflict, just factual "airlines currently route around this area."
var WatchZones = []Zone{
	{
		Name:      "Black Sea / Ukraine Airspace Advisory",
		CenterLat: 49.0,
		CenterLon: 32.0,
		RadiusKm:  500,
	},
	{
		Name:      "Red Sea Maritime & Air Risk Zone",
		CenterLat: 15.5,
		CenterLon: 41.5,
		RadiusKm:  450,
	},
	{
		Name:      "North Korea Restricted Airspace",
		CenterLat: 40.0,
		CenterLon: 127.5,
		RadiusKm:  300,
	},
	{
		Name:      "Strait of Hormuz Oil Chokepoint",
		CenterLat: 26.5,
		CenterLon: 56.3,
		RadiusKm:  200,
	},
	{
		Name:      "Suez Canal Trade Chokepoint",
		CenterLat: 30.5,
		CenterLon: 32.3,
		RadiusKm:  100,
	},
	{
		Name:      "Strait of Malacca Trade Chokepoint",
		CenterLat: 3.0,
		CenterLon: 100.5,
		RadiusKm:  150,
	},
}

type Zone struct {
	Name      string
	CenterLat float64
	CenterLon float64
	RadiusKm  float64
}

// haversineKm returns the great-circle distance between two lat/lon points.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func (z Zone) Contains(lat, lon float64) bool {
	return haversineKm(z.CenterLat, z.CenterLon, lat, lon) <= z.RadiusKm
}
