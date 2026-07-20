package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var upgrader = websocket.Upgrader{
	// Allow any origin for now — lock this down to your deployed frontend
	// domain before you consider this "production", but it's fine for a
	// portfolio project.
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	// .env is optional — on a real deploy (Railway etc.) you'll set these
	// as actual environment variables instead, so we ignore a missing file.
	_ = godotenv.Load()

	tm := NewTokenManager(os.Getenv("OPENSKY_CLIENT_ID"), os.Getenv("OPENSKY_CLIENT_SECRET"))
	if tm.Enabled() {
		log.Println("using authenticated OpenSky access")
	} else {
		log.Println("WARNING: no OPENSKY_CLIENT_ID/SECRET set — using anonymous access, which is heavily rate-limited and will likely 429")
	}
	if os.Getenv("GEMINI_API_KEY") == "" {
		log.Println("WARNING: no GEMINI_API_KEY set — /api/ask (Orbit Copilot) will return errors until you set one")
	}

	shipTracker := NewShipTracker()
	go RunAISStream(os.Getenv("AISSTREAM_API_KEY"), shipTracker)

	hub := NewHub()
	analyzer := NewAnalyzer()
	state := NewAppState()
	history := OpenHistory()

	go func() {
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ships := shipTracker.Snapshot()
			if len(ships) > 0 {
				state.SetShips(ships)
				hub.Broadcast("ships", map[string]interface{}{
					"type":  "ships",
					"ships": ships,
				})
			}
		}
	}()

	// Satellites are decoupled entirely from the flight data pipeline —
	// TLE data is fetched rarely (positions are pure local computation
	// after that), so satellite tracking keeps working smoothly even
	// during an OpenSky outage/throttle.
	satManager := NewSatelliteManager()
	if err := satManager.RefreshTLEs(); err != nil {
		log.Println("initial satellite TLE fetch failed (will retry):", err)
	}
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := satManager.RefreshTLEs(); err != nil {
				log.Println("satellite TLE refresh failed:", err)
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			hub.Broadcast("satellites", map[string]interface{}{
				"type":       "satellites",
				"satellites": satManager.Positions(time.Now().UTC()),
			})
		}
	}()

	// Weather changes slowly relative to flights/satellites, so this
	// refreshes far less often — free, keyless Open-Meteo grid data.
	go func() {
		fetchAndBroadcastWeather := func() {
			points, err := FetchWeatherGrid()
			if err != nil {
				log.Println("weather fetch error:", err)
			}
			if len(points) > 0 {
				var maxPrecip, maxWind float64
				activeCount := 0
				for _, p := range points {
					if p.PrecipitationMM > maxPrecip {
						maxPrecip = p.PrecipitationMM
					}
					if p.WindSpeedKmh > maxWind {
						maxWind = p.WindSpeedKmh
					}
					if p.PrecipitationMM > 0.1 || p.WindSpeedKmh > 35 {
						activeCount++
					}
				}
				log.Printf(
					"weather: %d points, %d showing active rain/wind, max precip %.1fmm, max wind %.0fkm/h\n",
					len(points), activeCount, maxPrecip, maxWind,
				)
				state.SetWeather(points)
				hub.Broadcast("weather", map[string]interface{}{
					"type":    "weather",
					"weather": points,
				})
			}
		}
		fetchAndBroadcastWeather()

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			fetchAndBroadcastWeather()
		}
	}()

	// Poll OpenSky every 12s, run the anomaly/zone analyzer over the fresh
	// snapshot, update shared state (used by the copilot), persist any new
	// insights to the history log, and broadcast to every connected client.
	go StartPolling(12*time.Second, tm, func(flights []Flight) {
		newInsights := analyzer.Process(flights)

		// Correlate zone presence + nearby weather + recent anomaly history
		// into one risk score per flight — this is the actual "fusion" step
		// that ties the separate live layers together into something worth
		// acting on, rather than four viewers you have to cross-reference
		// by eye.
		for i := range flights {
			f := &flights[i]
			insideZone := ""
			for _, z := range WatchZones {
				if z.Contains(f.Latitude, f.Longitude) {
					insideZone = z.Name
					break
				}
			}
			weatherPoint, hasWeather := state.NearestWeather(f.Latitude, f.Longitude)
			flagged, reason := analyzer.RecentlyFlagged(f.ICAO24, 10*time.Minute)
			f.RiskScore, f.RiskFactors = ComputeRisk(insideZone, weatherPoint, hasWeather, flagged, reason)
		}

		state.Update(flights, newInsights)
		history.Record(newInsights)

		hub.Broadcast("flights", map[string]interface{}{
			"type":     "flights",
			"time":     time.Now().Unix(),
			"flights":  flights,
			"insights": newInsights,
		})
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade error:", err)
			return
		}
		hub.Register(conn)
		log.Printf("client connected (total: %d)\n", hub.ClientCount())

		go func() {
			defer hub.Unregister(conn)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					log.Printf("client disconnected (total: %d)\n", hub.ClientCount()-1)
					return
				}
			}
		}()
	})

	// Orbit Copilot: answers natural-language questions grounded in the
	// live state snapshot (current flights, zone occupancy, recent
	// anomalies). This is a plain REST endpoint, separate from the
	// WebSocket feed, since it's a one-shot request/response.
	http.HandleFunc("/api/ask", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			Question string `json:"question"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Question) == "" {
			http.Error(w, "invalid request: expected {\"question\": \"...\"}", http.StatusBadRequest)
			return
		}

		answer, err := AskCopilot(body.Question, state)
		if err != nil {
			log.Println("copilot error:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"answer": answer})
	})

	http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history.Stats())
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("orbit backend listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
