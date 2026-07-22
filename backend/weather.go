package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type WeatherPoint struct {
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	PrecipitationMM float64 `json:"precipitationMm"`
	WindSpeedKmh    float64 `json:"windSpeedKmh"`
	TemperatureC    float64 `json:"temperatureC"`
}

type owmResponse struct {
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"` // m/s
	} `json:"wind"`
	Rain struct {
		OneHour float64 `json:"1h"`
	} `json:"rain"`
}

// weatherGrid returns the sample points we poll — a coarse lat/lon grid,
// skipping the poles.
func weatherGrid() [][2]float64 {
	var points [][2]float64
	for lat := -75.0; lat <= 75.0; lat += 15 {
		for lon := -180.0; lon < 180.0; lon += 40 {
			points = append(points, [2]float64{lat, lon})
		}
	}
	return points
}

// FetchWeatherGrid queries OpenWeatherMap's free Current Weather API once
// per grid point, paced to stay under the free tier's 60 calls/minute
// limit. Unlike Open-Meteo's keyless batch endpoint, this is keyed per
// account rather than pooled by IP — which matters on a platform like
// Railway where many unrelated apps can share an egress IP and exhaust a
// keyless per-IP quota collectively.
func FetchWeatherGrid(apiKey string) ([]WeatherPoint, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OPENWEATHER_API_KEY not set")
	}

	grid := weatherGrid()
	results := make([]WeatherPoint, 0, len(grid))
	client := &http.Client{Timeout: 10 * time.Second}
	failures := 0

	for _, p := range grid {
		url := fmt.Sprintf(
			"https://api.openweathermap.org/data/2.5/weather?lat=%.2f&lon=%.2f&appid=%s&units=metric",
			p[0], p[1], apiKey,
		)
		resp, err := client.Get(url)
		if err != nil {
			failures++
			time.Sleep(1100 * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			failures++
			if failures <= 3 { // avoid spamming the log if the key itself is bad
				log.Printf("openweathermap returned %d: %s\n", resp.StatusCode, truncate(string(body), 150))
			}
			time.Sleep(1100 * time.Millisecond)
			continue
		}

		var parsed owmResponse
		if json.Unmarshal(body, &parsed) != nil {
			time.Sleep(1100 * time.Millisecond)
			continue
		}

		results = append(results, WeatherPoint{
			Lat: p[0], Lon: p[1],
			PrecipitationMM: parsed.Rain.OneHour,
			WindSpeedKmh:    parsed.Wind.Speed * 3.6,
			TemperatureC:    parsed.Main.Temp,
		})
		time.Sleep(1100 * time.Millisecond)
	}

	if len(results) == 0 && failures > 0 {
		return nil, fmt.Errorf("all %d weather requests failed", failures)
	}
	return results, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
