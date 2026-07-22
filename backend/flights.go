package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Flight is the normalized shape we send to the frontend.
type Flight struct {
	ICAO24    string  `json:"icao24"`
	Callsign  string  `json:"callsign"`
	Country   string  `json:"originCountry"`
	Longitude float64 `json:"lon"`
	Latitude  float64 `json:"lat"`
	Altitude  float64 `json:"altitude"`
	Velocity  float64 `json:"velocity"`
	Heading   float64 `json:"heading"`
	OnGround  bool    `json:"onGround"`

	RiskScore   int      `json:"riskScore"`
	RiskFactors []string `json:"riskFactors,omitempty"`
}

// airplanes.live is a volunteer-run community ADS-B feed (same lineage as
// adsb.fi / ADSB Exchange) built specifically as an open alternative after
// OpenSky started restricting access from cloud/datacenter IPs. It's a
// point+radius API (no single "all aircraft" endpoint on the free tier),
// rate-limited to 1 request/second, no API key required.
const airplanesLiveBase = "https://api.airplanes.live/v2/point"
const queryRadiusNM = 250

type queryPoint struct {
	Name     string
	Lat, Lon float64
}

// Query points combine our trade-chokepoint zones with major aviation
// hubs, so the globe shows meaningful traffic clusters rather than a
// handful of scattered dots — full uniform global coverage isn't
// available from a free point-radius API without a paid tier.
func queryPoints() []queryPoint {
	points := []queryPoint{
		{"New York", 40.7, -74.0},
		{"London", 51.5, -0.1},
		{"Frankfurt", 50.1, 8.7},
		{"Dubai", 25.2, 55.3},
		{"Singapore", 1.35, 103.8},
		{"Tokyo", 35.7, 139.7},
		{"Mumbai", 19.1, 72.9},
		{"Los Angeles", 34.0, -118.2},
	}
	for _, z := range WatchZones {
		points = append(points, queryPoint{z.Name, z.CenterLat, z.CenterLon})
	}
	return points
}

type FlightTracker struct {
	mu      sync.RWMutex
	flights map[string]Flight
}

func NewFlightTracker() *FlightTracker {
	return &FlightTracker{flights: make(map[string]Flight)}
}

func (t *FlightTracker) update(f Flight) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flights[f.ICAO24] = f
}

func (t *FlightTracker) Snapshot() []Flight {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]Flight, 0, len(t.flights))
	for _, f := range t.flights {
		out = append(out, f)
	}
	return out
}

type airplanesLiveResponse struct {
	Ac []struct {
		Hex     string          `json:"hex"`
		Flight  string          `json:"flight"`
		Lat     float64         `json:"lat"`
		Lon     float64         `json:"lon"`
		AltBaro json.RawMessage `json:"alt_baro"` // number, or the string "ground"
		Gs      float64         `json:"gs"`       // knots
		Track   float64         `json:"track"`    // degrees
	} `json:"ac"`
}

func fetchPoint(p queryPoint) ([]Flight, error) {
	url := fmt.Sprintf("%s/%.4f/%.4f/%d", airplanesLiveBase, p.Lat, p.Lon, queryRadiusNM)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "orbit-tracker/1.0 (portfolio project)")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("airplanes.live returned %d: %s", resp.StatusCode, truncate(string(body), 150))
	}

	var parsed airplanesLiveResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	flights := make([]Flight, 0, len(parsed.Ac))
	for _, ac := range parsed.Ac {
		if ac.Hex == "" || (ac.Lat == 0 && ac.Lon == 0) {
			continue
		}

		var altitudeM float64
		var onGround bool
		var altStr string
		if json.Unmarshal(ac.AltBaro, &altStr) == nil {
			onGround = strings.EqualFold(altStr, "ground")
		} else {
			var altFt float64
			if json.Unmarshal(ac.AltBaro, &altFt) == nil {
				altitudeM = altFt * 0.3048
			}
		}

		flights = append(flights, Flight{
			ICAO24:    strings.ToUpper(ac.Hex),
			Callsign:  strings.TrimSpace(ac.Flight),
			Longitude: ac.Lon,
			Latitude:  ac.Lat,
			Altitude:  altitudeM,
			Velocity:  ac.Gs * 0.514444, // knots -> m/s, matches downstream unit assumptions
			Heading:   ac.Track,
			OnGround:  onGround,
		})
	}
	return flights, nil
}

// RunFlightTracking sweeps through query points forever, pacing requests
// to respect the 1 request/second limit, keeping the tracker updated with
// the latest known position for every aircraft seen. Decoupled from the
// analysis/broadcast cycle in main.go on purpose — this loop's only job
// is "keep fresh data flowing."
func RunFlightTracking(tracker *FlightTracker) {
	points := queryPoints()
	consecutiveFailures := 0

	for {
		anySuccess := false
		for _, p := range points {
			flights, err := fetchPoint(p)
			if err != nil {
				log.Printf("airplanes.live fetch error (%s): %v\n", p.Name, err)
			} else {
				anySuccess = true
				for _, f := range flights {
					tracker.update(f)
				}
			}
			time.Sleep(1100 * time.Millisecond)
		}

		if anySuccess {
			consecutiveFailures = 0
		} else {
			consecutiveFailures++
			backoff := time.Duration(min(consecutiveFailures, 10)) * 10 * time.Second
			log.Printf("all airplanes.live queries failed, backing off %s\n", backoff)
			time.Sleep(backoff)
		}
	}
}
