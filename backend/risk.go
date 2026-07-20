package main

import "fmt"

// ComputeRisk fuses three independently-tracked signals — watch-zone
// presence, nearby weather severity, and recent anomaly history — into
// one score. This is the actual point of collecting all this live data
// in one system rather than four separate viewers: correlating them is
// what turns raw feeds into something worth acting on.
func ComputeRisk(insideZone string, weather WeatherPoint, hasWeather bool, anomalyFlagged bool, anomalyReason string) (int, []string) {
	score := 0
	var factors []string

	if insideZone != "" {
		score += 40
		factors = append(factors, "inside watch zone: "+insideZone)
	}
	if hasWeather && (weather.PrecipitationMM > 2 || weather.WindSpeedKmh > 50) {
		score += 30
		factors = append(factors, fmt.Sprintf(
			"active weather nearby (%.1fmm rain, %.0fkm/h wind)", weather.PrecipitationMM, weather.WindSpeedKmh,
		))
	}
	if anomalyFlagged {
		score += 30
		factors = append(factors, "recent anomaly: "+anomalyReason)
	}
	if score > 100 {
		score = 100
	}
	return score, factors
}
