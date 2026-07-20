package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Open-Meteo's forecast API is genuinely free and keyless, and supports
// batching many lat/lon pairs into one request via comma-separated lists.
const weatherBaseURL = "https://api.open-meteo.com/v1/forecast"

type WeatherPoint struct {
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	PrecipitationMM float64 `json:"precipitationMm"`
	WindSpeedKmh    float64 `json:"windSpeedKmh"`
	TemperatureC    float64 `json:"temperatureC"`
}

type openMeteoResult struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Current   struct {
		Precipitation float64 `json:"precipitation"`
		WindSpeed10m  float64 `json:"wind_speed_10m"`
		Temperature2m float64 `json:"temperature_2m"`
	} `json:"current"`
}

// weatherGrid returns the sample points we poll — a coarse lat/lon grid,
// skipping the poles where satellite-visualization value is low and grid
// cells get visually crowded from equirectangular distortion.
func weatherGrid() [][2]float64 {
	var points [][2]float64
	for lat := -75.0; lat <= 75.0; lat += 15 {
		for lon := -180.0; lon < 180.0; lon += 20 {
			points = append(points, [2]float64{lat, lon})
		}
	}
	return points
}

// FetchWeatherGrid queries Open-Meteo in chunks (its URL-length-friendly
// batch limit) and returns current precipitation/wind for the whole grid.
func FetchWeatherGrid() ([]WeatherPoint, error) {
	grid := weatherGrid()
	const chunkSize = 40
	var results []WeatherPoint

	for i := 0; i < len(grid); i += chunkSize {
		end := i + chunkSize
		if end > len(grid) {
			end = len(grid)
		}
		chunk := grid[i:end]

		lats := make([]string, len(chunk))
		lons := make([]string, len(chunk))
		for j, p := range chunk {
			lats[j] = fmt.Sprintf("%.2f", p[0])
			lons[j] = fmt.Sprintf("%.2f", p[1])
		}

		url := fmt.Sprintf("%s?latitude=%s&longitude=%s&current=precipitation,wind_speed_10m,temperature_2m",
			weatherBaseURL, strings.Join(lats, ","), strings.Join(lons, ","))

		points, err := fetchWeatherChunk(url)
		if err != nil {
			return results, err // return what we have so far plus the error
		}
		results = append(results, points...)
	}

	return results, nil
}

func fetchWeatherChunk(url string) ([]WeatherPoint, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Open-Meteo returns a single object for one location, or a JSON array
	// for multiple — since we always send multiple, we expect an array.
	var raw []openMeteoResult
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse Open-Meteo response: %w (%s)", err, truncate(string(body), 200))
	}

	points := make([]WeatherPoint, 0, len(raw))
	for _, r := range raw {
		points = append(points, WeatherPoint{
			Lat: r.Latitude, Lon: r.Longitude,
			PrecipitationMM: r.Current.Precipitation,
			WindSpeedKmh:    r.Current.WindSpeed10m,
			TemperatureC:    r.Current.Temperature2m,
		})
	}
	return points, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
