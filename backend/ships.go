package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const aisStreamURL = "wss://stream.aisstream.io/v0/stream"

type Ship struct {
	MMSI      string  `json:"mmsi"`
	Name      string  `json:"name"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	SpeedKn   float64 `json:"speedKn"`
	CourseDeg float64 `json:"courseDeg"`
}

type ShipTracker struct {
	mu    sync.RWMutex
	ships map[string]shipEntry
}

type shipEntry struct {
	ship      Ship
	updatedAt time.Time
}

func NewShipTracker() *ShipTracker {
	return &ShipTracker{ships: make(map[string]shipEntry)}
}

func (t *ShipTracker) update(s Ship) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ships[s.MMSI] = shipEntry{ship: s, updatedAt: time.Now()}
}

// Snapshot returns currently known ships, dropping any not updated in the
// last 20 minutes — AIS-equipped vessels report frequently while
// underway, so a long silence usually means it's out of range of a
// receiver, not that it teleported away.
func (t *ShipTracker) Snapshot() []Ship {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cutoff := time.Now().Add(-20 * time.Minute)
	ships := make([]Ship, 0, len(t.ships))
	for _, e := range t.ships {
		if e.updatedAt.After(cutoff) {
			ships = append(ships, e.ship)
		}
	}
	return ships
}

type aisSubscription struct {
	APIKey             string         `json:"APIKey"`
	BoundingBoxes      [][][2]float64 `json:"BoundingBoxes"`
	FilterMessageTypes []string       `json:"FilterMessageTypes"`
}

type aisEnvelope struct {
	MessageType string `json:"MessageType"`
	MetaData    struct {
		MMSI      json.Number `json:"MMSI"`
		ShipName  string      `json:"ShipName"`
		Latitude  float64     `json:"latitude"`
		Longitude float64     `json:"longitude"`
	} `json:"MetaData"`
	Message struct {
		PositionReport struct {
			Sog float64 `json:"Sog"`
			Cog float64 `json:"Cog"`
		} `json:"PositionReport"`
	} `json:"Message"`
}

// boundingBoxesFromZones builds AIS subscription bounding boxes around
// each watch zone, rather than subscribing to the whole planet — aisstream
// warns global subscriptions run ~300 msg/s and need serious bandwidth,
// and we only actually care about traffic through named trade chokepoints
// anyway.
func boundingBoxesFromZones() [][][2]float64 {
	var boxes [][][2]float64
	for _, z := range WatchZones {
		degLat := z.RadiusKm / 111.0
		degLon := z.RadiusKm / (111.0 * cosApprox(z.CenterLat))
		boxes = append(boxes, [][2]float64{
			{z.CenterLat - degLat, z.CenterLon - degLon},
			{z.CenterLat + degLat, z.CenterLon + degLon},
		})
	}
	return boxes
}

func cosApprox(latDeg float64) float64 {
	rad := latDeg * 3.14159265 / 180
	c := 1.0 - rad*rad/2 // fine approximation at these latitudes
	if c < 0.2 {
		c = 0.2 // guard against div-by-near-zero close to the poles
	}
	return c
}

// RunAISStream connects to aisstream.io and feeds every position report
// into the tracker, forever, reconnecting with backoff on any failure —
// same resilience shape as the OpenSky poller, just event-driven instead
// of polled.
func RunAISStream(apiKey string, tracker *ShipTracker) {
	if apiKey == "" {
		log.Println("WARNING: no AISSTREAM_API_KEY set — ship tracking disabled")
		return
	}

	backoff := 5 * time.Second
	const maxBackoff = 2 * time.Minute

	for {
		if err := connectAndStream(apiKey, tracker); err != nil {
			log.Println("aisstream connection error:", err)
		}
		log.Printf("aisstream disconnected, reconnecting in %s\n", backoff)
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func connectAndStream(apiKey string, tracker *ShipTracker) error {
	conn, _, err := websocket.DefaultDialer.Dial(aisStreamURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	sub := aisSubscription{
		APIKey:             apiKey,
		BoundingBoxes:      boundingBoxesFromZones(),
		FilterMessageTypes: []string{"PositionReport"},
	}
	if err := conn.WriteJSON(sub); err != nil {
		return err
	}
	log.Println("aisstream connected and subscribed to trade-chokepoint zones")

	for {
		var env aisEnvelope
		if err := conn.ReadJSON(&env); err != nil {
			return err
		}
		if env.MessageType != "PositionReport" {
			continue
		}
		tracker.update(Ship{
			MMSI:      env.MetaData.MMSI.String(),
			Name:      env.MetaData.ShipName,
			Lat:       env.MetaData.Latitude,
			Lon:       env.MetaData.Longitude,
			SpeedKn:   env.Message.PositionReport.Sog,
			CourseDeg: env.Message.PositionReport.Cog,
		})
	}
}
