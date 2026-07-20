package main

import (
	"bufio"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CelesTrak's "visual" group — roughly the 100-200 brightest, most
// commonly tracked satellites (ISS, Hubble, Starlink batches, etc.).
// Free, no API key, no signup — TLE data is published openly.
const tleSourceURL = "https://celestrak.org/NORAD/elements/gp.php?GROUP=visual&FORMAT=tle"

type SatellitePosition struct {
	Name       string  `json:"name"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	AltitudeKm float64 `json:"altitudeKm"`
}

type SatelliteManager struct {
	mu       sync.RWMutex
	elements []*OrbitalElements
}

func NewSatelliteManager() *SatelliteManager {
	return &SatelliteManager{}
}

// RefreshTLEs fetches the current TLE set. TLEs are only republished a
// few times a day, so this only needs to run occasionally (unlike the
// flight poller) — the actual position math runs locally and can be
// recomputed as often as we like without hitting the network again.
func (m *SatelliteManager) RefreshTLEs() error {
	req, err := http.NewRequest(http.MethodGet, tleSourceURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "orbit-tracker/1.0 (portfolio project)")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var elements []*OrbitalElements
	scanner := bufio.NewScanner(resp.Body)
	var name, line1 string
	lineNum := 0

	for scanner.Scan() {
		text := strings.TrimRight(scanner.Text(), "\r\n")
		switch lineNum % 3 {
		case 0:
			name = text
		case 1:
			line1 = text
		case 2:
			oe, err := ParseTLE(name, line1, text)
			if err == nil {
				elements = append(elements, oe)
			}
		}
		lineNum++
	}

	if len(elements) == 0 {
		return err // scanner error, or genuinely empty response
	}

	m.mu.Lock()
	m.elements = elements
	m.mu.Unlock()

	log.Printf("loaded %d satellite TLEs\n", len(elements))
	return nil
}

// Positions propagates every tracked satellite to the given time. This is
// pure computation (no network call), so it's cheap enough to call every
// few seconds for smooth motion.
func (m *SatelliteManager) Positions(t time.Time) []SatellitePosition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	positions := make([]SatellitePosition, 0, len(m.elements))
	for _, oe := range m.elements {
		lat, lon, alt := Propagate(oe, t)
		positions = append(positions, SatellitePosition{
			Name: oe.Name, Lat: lat, Lon: lon, AltitudeKm: alt,
		})
	}
	return positions
}
