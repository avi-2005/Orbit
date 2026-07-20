package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type Insight struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`     // "holding" | "descent" | "zone"
	Severity string  `json:"severity"` // "info" | "warning" | "alert"
	ICAO24   string  `json:"icao24"`
	Callsign string  `json:"callsign"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Altitude float64 `json:"altitude"`
	Detail   string  `json:"detail"`
	Time     int64   `json:"time"`
}

type sample struct {
	t        time.Time
	lat, lon float64
	altitude float64
	heading  float64
}

type aircraftTrack struct {
	samples []sample

	lastHoldingAlert time.Time
	lastDescentAlert time.Time
	insideZone       map[string]bool // zone name -> currently inside
}

const (
	maxSamples      = 6
	alertCooldown   = 5 * time.Minute
	holdingDegrees  = 250.0 // cumulative heading swing to call it "circling"
	holdingAltBandM = 200.0 // altitude must stay within this band
	holdingRadiusKm = 15.0  // and stay roughly in one place
	descentRateMS   = 15.0  // vertical rate (m/s) to flag as "rapid"
	descentDropM    = 300.0 // minimum altitude drop to avoid noise
)

// Analyzer holds rolling state across poll cycles. It's intentionally
// single-goroutine (only the poller calls Process), so no locking is
// needed internally — but it's exposed behind a mutex anyway in case a
// future HTTP endpoint wants to inspect current state without racing.
type Analyzer struct {
	mu     sync.Mutex
	tracks map[string]*aircraftTrack
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{tracks: make(map[string]*aircraftTrack)}
}

// Process ingests one poll's worth of flights and returns any newly
// detected insights (holding patterns, rapid descents, zone entries).
// Each insight type has its own cooldown/transition logic so we emit a
// notification once per episode, not once per poll.
func (a *Analyzer) Process(flights []Flight) []Insight {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	var insights []Insight
	seen := make(map[string]bool, len(flights))

	for _, f := range flights {
		if f.ICAO24 == "" || f.OnGround {
			continue
		}
		seen[f.ICAO24] = true

		track, ok := a.tracks[f.ICAO24]
		if !ok {
			track = &aircraftTrack{insideZone: make(map[string]bool)}
			a.tracks[f.ICAO24] = track
		}

		track.samples = append(track.samples, sample{
			t: now, lat: f.Latitude, lon: f.Longitude,
			altitude: f.Altitude, heading: f.Heading,
		})
		if len(track.samples) > maxSamples {
			track.samples = track.samples[len(track.samples)-maxSamples:]
		}

		if ins, ok := detectHolding(f, track, now); ok {
			insights = append(insights, ins)
		}
		if ins, ok := detectDescent(f, track, now); ok {
			insights = append(insights, ins)
		}
		insights = append(insights, detectZoneEntries(f, track)...)
	}

	// Forget aircraft we haven't seen in this snapshot so memory doesn't
	// grow unbounded over a long-running server process.
	for icao := range a.tracks {
		if !seen[icao] {
			delete(a.tracks, icao)
		}
	}

	return insights
}

// RecentlyFlagged reports whether an aircraft has had a holding-pattern or
// rapid-descent alert within the given window — used to fold anomaly
// history into the risk score without re-deriving it from scratch.
func (a *Analyzer) RecentlyFlagged(icao24 string, window time.Duration) (bool, string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	track, ok := a.tracks[icao24]
	if !ok {
		return false, ""
	}
	now := time.Now()
	if now.Sub(track.lastHoldingAlert) < window {
		return true, "holding pattern"
	}
	if now.Sub(track.lastDescentAlert) < window {
		return true, "rapid descent"
	}
	return false, ""
}

func detectHolding(f Flight, track *aircraftTrack, now time.Time) (Insight, bool) {
	if len(track.samples) < 4 || now.Sub(track.lastHoldingAlert) < alertCooldown {
		return Insight{}, false
	}

	var headingSwing float64
	minAlt, maxAlt := math.Inf(1), math.Inf(-1)
	for i := 1; i < len(track.samples); i++ {
		headingSwing += math.Abs(angularDiff(track.samples[i-1].heading, track.samples[i].heading))
	}
	for _, s := range track.samples {
		minAlt = math.Min(minAlt, s.altitude)
		maxAlt = math.Max(maxAlt, s.altitude)
	}

	first := track.samples[0]
	spreadKm := haversineKm(first.lat, first.lon, f.Latitude, f.Longitude)

	if headingSwing >= holdingDegrees && (maxAlt-minAlt) <= holdingAltBandM && spreadKm <= holdingRadiusKm {
		track.lastHoldingAlert = now
		return Insight{
			ID:       fmt.Sprintf("%s-holding-%d", f.ICAO24, now.Unix()),
			Type:     "holding",
			Severity: "warning",
			ICAO24:   f.ICAO24,
			Callsign: f.Callsign,
			Lat:      f.Latitude,
			Lon:      f.Longitude,
			Altitude: f.Altitude,
			Detail:   "Aircraft appears to be circling in a holding pattern",
			Time:     now.Unix(),
		}, true
	}
	return Insight{}, false
}

func detectDescent(f Flight, track *aircraftTrack, now time.Time) (Insight, bool) {
	if len(track.samples) < 2 || now.Sub(track.lastDescentAlert) < alertCooldown {
		return Insight{}, false
	}

	prev := track.samples[len(track.samples)-2]
	cur := track.samples[len(track.samples)-1]
	dt := cur.t.Sub(prev.t).Seconds()
	if dt <= 0 {
		return Insight{}, false
	}

	drop := prev.altitude - cur.altitude
	rate := drop / dt

	if rate >= descentRateMS && drop >= descentDropM {
		track.lastDescentAlert = now
		return Insight{
			ID:       fmt.Sprintf("%s-descent-%d", f.ICAO24, now.Unix()),
			Type:     "descent",
			Severity: "alert",
			ICAO24:   f.ICAO24,
			Callsign: f.Callsign,
			Lat:      f.Latitude,
			Lon:      f.Longitude,
			Altitude: f.Altitude,
			Detail:   fmt.Sprintf("Rapid descent: ~%.0fm lost in %.0fs", drop, dt),
			Time:     now.Unix(),
		}, true
	}
	return Insight{}, false
}

func detectZoneEntries(f Flight, track *aircraftTrack) []Insight {
	var insights []Insight
	for _, z := range WatchZones {
		inside := z.Contains(f.Latitude, f.Longitude)
		wasInside := track.insideZone[z.Name]

		if inside && !wasInside {
			insights = append(insights, Insight{
				ID:       fmt.Sprintf("%s-zone-%s-%d", f.ICAO24, z.Name, time.Now().Unix()),
				Type:     "zone",
				Severity: "info",
				ICAO24:   f.ICAO24,
				Callsign: f.Callsign,
				Lat:      f.Latitude,
				Lon:      f.Longitude,
				Altitude: f.Altitude,
				Detail:   fmt.Sprintf("Entered watch zone: %s", z.Name),
				Time:     time.Now().Unix(),
			})
		}
		track.insideZone[z.Name] = inside
	}
	return insights
}

// angularDiff returns the signed shortest angular distance from a to b in
// degrees, correctly handling the 0/360 wraparound.
func angularDiff(a, b float64) float64 {
	d := math.Mod(b-a+540, 360) - 180
	return d
}
