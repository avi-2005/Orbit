package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Flight is the normalized shape we send to the frontend. OpenSky's raw
// "state vector" format is a positional array, which is efficient over
// the wire but painful to work with in JS, so we translate it here.
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

	// Populated after the analyzer runs, not from raw OpenSky data — see
	// risk.go. Zero value (0, nil) means "not yet scored."
	RiskScore   int      `json:"riskScore"`
	RiskFactors []string `json:"riskFactors,omitempty"`
}

type openSkyResponse struct {
	Time   int64           `json:"time"`
	States [][]interface{} `json:"states"`
}

const openSkyURL = "https://opensky-network.org/api/states/all"

// FetchFlights hits the OpenSky endpoint and returns normalized flights.
// If tm is non-nil and has credentials configured, requests are
// authenticated (4,000 requests/day) — otherwise we fall back to the
// anonymous tier, which shares a much smaller daily quota across everyone
// hitting OpenSky from a similar network range and will 429 quickly.
func FetchFlights(tm *TokenManager) ([]Flight, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, openSkyURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "orbit-tracker/1.0 (portfolio project)")

	if tm != nil && tm.Enabled() {
		token, err := tm.Token()
		if err != nil {
			// Don't abort the whole fetch just because the auth server is
			// unreachable — auth.opensky-network.org and
			// opensky-network.org are different hosts, so one being down
			// (or blocked) doesn't necessarily mean the other is. Fall
			// back to an anonymous request rather than returning nothing.
			log.Printf("auth token fetch failed, falling back to anonymous request: %v\n", err)
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

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
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("opensky returned %d: %s", resp.StatusCode, snippet)
	}

	var parsed openSkyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	flights := make([]Flight, 0, len(parsed.States))
	for _, s := range parsed.States {
		f, ok := parseState(s)
		if !ok {
			continue
		}
		flights = append(flights, f)
	}
	return flights, nil
}

// parseState converts one raw OpenSky state vector (a loosely-typed array)
// into a Flight. Many fields can be null (e.g. altitude for a plane still
// on the ground), so we guard every index.
func parseState(s []interface{}) (Flight, bool) {
	if len(s) < 11 {
		return Flight{}, false
	}

	lon, lonOK := s[5].(float64)
	lat, latOK := s[6].(float64)
	if !lonOK || !latOK {
		// No position fix yet — skip, nothing to plot on the globe.
		return Flight{}, false
	}

	icao24, _ := s[0].(string)
	callsign, _ := s[1].(string)
	country, _ := s[2].(string)
	altitude, _ := s[7].(float64)
	onGround, _ := s[8].(bool)
	velocity, _ := s[9].(float64)
	heading, _ := s[10].(float64)

	return Flight{
		ICAO24:    icao24,
		Callsign:  callsign,
		Country:   country,
		Longitude: lon,
		Latitude:  lat,
		Altitude:  altitude,
		Velocity:  velocity,
		Heading:   heading,
		OnGround:  onGround,
	}, true
}

// StartPolling runs forever, fetching fresh flight data on an interval and
// handing it to the callback (which the caller wires up to Hub.Broadcast).
//
// On failure it backs off (interval * consecutive failure count, capped),
// instead of retrying on a fixed schedule. Hitting an already-struggling
// or rate-limited server on a rigid timer tends to prolong the throttling
// rather than recover from it — backing off gives it room to recover.
func StartPolling(interval time.Duration, tm *TokenManager, onUpdate func([]Flight)) {
	const maxBackoff = 3 * time.Minute
	consecutiveFailures := 0

	for {
		flights, err := FetchFlights(tm)

		if err != nil {
			consecutiveFailures++
			log.Println("opensky fetch error:", err)
		} else {
			if consecutiveFailures > 0 {
				log.Printf("opensky recovered after %d consecutive failures\n", consecutiveFailures)
			}
			consecutiveFailures = 0
			log.Printf("polled %d flights\n", len(flights))
			onUpdate(flights)
		}

		wait := interval
		if consecutiveFailures > 0 {
			backoff := interval * time.Duration(consecutiveFailures)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			wait = backoff
			log.Printf("backing off %s before next attempt (%d consecutive failures)\n", wait, consecutiveFailures)
		}
		time.Sleep(wait)
	}
}
