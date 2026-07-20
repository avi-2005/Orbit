package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

const maxStoredInsights = 50

// AppState holds the latest known flights plus a rolling history of
// flagged insights. Snapshot() turns this into a compact text block that
// gets handed to the LLM as grounding context — the copilot never guesses
// about live conditions, it only summarizes what's actually in here.
type AppState struct {
	mu       sync.RWMutex
	flights  []Flight
	insights []Insight // newest first
	weather  []WeatherPoint
	ships    []Ship
}

func NewAppState() *AppState {
	return &AppState{}
}

func (s *AppState) SetWeather(points []WeatherPoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weather = points
}

// NearestWeather finds the closest sampled grid point to a given
// position — the grid is coarse (points every 15-20°), so this is a
// nearby reading, not an exact one at the flight's location.
func (s *AppState) NearestWeather(lat, lon float64) (WeatherPoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.weather) == 0 {
		return WeatherPoint{}, false
	}
	best := s.weather[0]
	bestDist := haversineKm(lat, lon, best.Lat, best.Lon)
	for _, w := range s.weather[1:] {
		d := haversineKm(lat, lon, w.Lat, w.Lon)
		if d < bestDist {
			bestDist = d
			best = w
		}
	}
	return best, true
}

func (s *AppState) Update(flights []Flight, newInsights []Insight) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flights = flights
	if len(newInsights) > 0 {
		s.insights = append(newInsights, s.insights...)
		if len(s.insights) > maxStoredInsights {
			s.insights = s.insights[:maxStoredInsights]
		}
	}
}

// SearchFlights filters currently tracked flights by registration country
// and/or callsign substring (case-insensitive). Used by the copilot's
// search_flights tool so it can answer specific questions instead of only
// summarizing.
func (s *AppState) SearchFlights(originCountry, callsignSubstr string, limit int) []Flight {
	s.mu.RLock()
	defer s.mu.RUnlock()

	country := strings.ToLower(strings.TrimSpace(originCountry))
	callsign := strings.ToLower(strings.TrimSpace(callsignSubstr))

	var results []Flight
	for _, f := range s.flights {
		if country != "" && !strings.Contains(strings.ToLower(f.Country), country) {
			continue
		}
		if callsign != "" && !strings.Contains(strings.ToLower(f.Callsign), callsign) {
			continue
		}
		results = append(results, f)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

// HighRiskFlights returns currently tracked flights with a risk score at
// or above minScore, sorted highest-risk first.
func (s *AppState) HighRiskFlights(minScore, limit int) []Flight {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []Flight
	for _, f := range s.flights {
		if f.RiskScore >= minScore {
			results = append(results, f)
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].RiskScore > results[j].RiskScore })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

func (s *AppState) SetShips(ships []Ship) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ships = ships
}

// ChokepointCongestion is the actual "trade intelligence" signal this
// whole project is aimed at: how many ships and flights are currently
// moving through each named chokepoint zone right now. A real trading
// desk watching e.g. Hormuz cares about exactly this kind of count.
type ChokepointCongestion struct {
	Zone        string `json:"zone"`
	ShipCount   int    `json:"shipCount"`
	FlightCount int    `json:"flightCount"`
}

func (s *AppState) ChokepointCongestion() []ChokepointCongestion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.chokepointCongestionLocked()
}

// chokepointCongestionLocked assumes the caller already holds a read (or
// write) lock — used internally by Snapshot() to avoid re-acquiring
// RLock, which is unsafe to nest if a writer happens to be waiting.
func (s *AppState) chokepointCongestionLocked() []ChokepointCongestion {
	result := make([]ChokepointCongestion, 0, len(WatchZones))
	for _, z := range WatchZones {
		var c ChokepointCongestion
		c.Zone = z.Name
		for _, sh := range s.ships {
			if z.Contains(sh.Lat, sh.Lon) {
				c.ShipCount++
			}
		}
		for _, f := range s.flights {
			if z.Contains(f.Latitude, f.Longitude) {
				c.FlightCount++
			}
		}
		result = append(result, c)
	}
	return result
}

func (s *AppState) Snapshot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Total aircraft currently tracked: %d\n\n", len(s.flights)))

	zoneCounts := make(map[string]int, len(WatchZones))
	for _, f := range s.flights {
		for _, z := range WatchZones {
			if z.Contains(f.Latitude, f.Longitude) {
				zoneCounts[z.Name]++
			}
		}
	}
	sb.WriteString("Aircraft currently inside each watch zone:\n")
	for _, z := range WatchZones {
		sb.WriteString(fmt.Sprintf("- %s: %d aircraft\n", z.Name, zoneCounts[z.Name]))
	}

	highRisk := 0
	for _, f := range s.flights {
		if f.RiskScore >= 50 {
			highRisk++
		}
	}
	sb.WriteString(fmt.Sprintf("\nFlights currently at elevated risk (score >= 50, combining zone/weather/anomaly signals): %d\n", highRisk))

	sb.WriteString("\nChokepoint congestion (ships and flights currently in each zone):\n")
	for _, c := range s.chokepointCongestionLocked() {
		sb.WriteString(fmt.Sprintf("- %s: %d ships, %d flights\n", c.Zone, c.ShipCount, c.FlightCount))
	}

	sb.WriteString("\nRecent flagged events (most recent first):\n")
	limit := len(s.insights)
	if limit > 15 {
		limit = 15
	}
	if limit == 0 {
		sb.WriteString("(none in the current session yet)\n")
	}
	for i := 0; i < limit; i++ {
		ins := s.insights[i]
		sb.WriteString(fmt.Sprintf(
			"- [%s/%s] callsign=%s icao24=%s — %s\n",
			ins.Type, ins.Severity, strings.TrimSpace(ins.Callsign), ins.ICAO24, ins.Detail,
		))
	}

	return sb.String()
}
