package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const geminiModel = "gemini-flash-latest"

func geminiURL(apiKey string) string {
	return fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		geminiModel, apiKey,
	)
}

type geminiRequest struct {
	SystemInstruction geminiContent   `json:"systemInstruction"`
	Contents          []geminiContent `json:"contents"`
	Tools             []geminiTool    `json:"tools,omitempty"`
	GenerationConfig  geminiGenConfig `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

// A part is a union type in the Gemini API — exactly one of these fields
// is populated depending on whether it's plain text, a tool invocation
// the model wants to make, or our result from having run that tool.
type geminiPart struct {
	Text             string                `json:"text,omitempty"`
	FunctionCall     *functionCall         `json:"functionCall,omitempty"`
	FunctionResponse *functionResponsePart `json:"functionResponse,omitempty"`
	// Gemini 3.x "thinking" models attach an opaque signature to function
	// call parts that must be echoed back unmodified on the next turn, or
	// the API rejects the follow-up request. We don't interpret it, just
	// carry it through.
	ThoughtSignature string `json:"thoughtSignature,omitempty"`
}

type functionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type functionResponsePart struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []functionDeclaration `json:"functionDeclarations"`
}

type functionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type geminiGenConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func searchFlightsTool() geminiTool {
	return geminiTool{
		FunctionDeclarations: []functionDeclaration{
			{
				Name: "search_flights",
				Description: "Search currently tracked live flights by callsign substring " +
					"(e.g. an airline code like 'AI' for Air India, or 'UAL' for United). " +
					"Note: this data source does not provide registration country — only " +
					"callsign search is meaningful.",
				Parameters: map[string]interface{}{
					"type": "OBJECT",
					"properties": map[string]interface{}{
						"callsignContains": map[string]interface{}{
							"type":        "STRING",
							"description": "Substring to match against flight callsigns.",
						},
						"limit": map[string]interface{}{
							"type":        "INTEGER",
							"description": "Max results to return, default 10, max 25.",
						},
					},
				},
			},
		},
	}
}

func chokepointCongestionTool() geminiTool {
	return geminiTool{
		FunctionDeclarations: []functionDeclaration{
			{
				Name: "get_chokepoint_congestion",
				Description: "Get the current number of ships and flights inside each named " +
					"trade chokepoint zone (Strait of Hormuz, Suez Canal, Strait of Malacca, " +
					"Red Sea, Black Sea, North Korea airspace). Use this for any question about " +
					"maritime traffic, oil chokepoints, trade routes, or shipping congestion.",
				Parameters: map[string]interface{}{"type": "OBJECT", "properties": map[string]interface{}{}},
			},
		},
	}
}

func highRiskFlightsTool() geminiTool {
	return geminiTool{
		FunctionDeclarations: []functionDeclaration{
			{
				Name: "get_high_risk_flights",
				Description: "Get currently tracked flights with an elevated risk score — a " +
					"correlated signal combining watch-zone presence, nearby severe weather, " +
					"and recent anomaly (holding pattern/rapid descent) history. Use this " +
					"whenever the user asks about risk, danger, concerning flights, or 'what " +
					"should I be watching right now.'",
				Parameters: map[string]interface{}{
					"type": "OBJECT",
					"properties": map[string]interface{}{
						"minScore": map[string]interface{}{
							"type":        "INTEGER",
							"description": "Minimum risk score 0-100 to include, default 50.",
						},
						"limit": map[string]interface{}{
							"type":        "INTEGER",
							"description": "Max results, default 10.",
						},
					},
				},
			},
		},
	}
}

// AskCopilot answers a natural-language question about live airspace.
// Rather than stuffing everything into one static prompt, the model is
// given a search_flights tool it can call whenever it needs specifics —
// this is the standard function-calling / tool-use loop: ask the model,
// if it requests a tool call, run it locally, feed the result back, and
// let the model produce its final answer grounded in that real result.
func AskCopilot(question string, state *AppState) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not set on the server")
	}

	system := "You are Orbit Copilot, an assistant embedded in a live flight-tracking " +
		"dashboard called Orbit. You have a search_flights tool — use it whenever the user " +
		"asks about specific flights, airlines, or countries, rather than saying you lack " +
		"the data. For general questions about watch-zone occupancy or recent anomalies, " +
		"use the summary below; it's already current.\n\n" +
		"LIVE SUMMARY:\n" + state.Snapshot() +
		"\n\nBe concise (2-4 sentences) and never invent flights or numbers not returned by " +
		"the tool or present in the summary."

	contents := []geminiContent{
		{Role: "user", Parts: []geminiPart{{Text: question}}},
	}
	tools := []geminiTool{searchFlightsTool(), highRiskFlightsTool(), chokepointCongestionTool()}

	for attempt := 0; attempt < 3; attempt++ {
		parts, err := callGemini(apiKey, system, contents, tools)
		if err != nil {
			return "", err
		}

		var call *functionCall
		var signature string
		var text strings.Builder
		for _, p := range parts {
			if p.FunctionCall != nil {
				call = p.FunctionCall
				signature = p.ThoughtSignature
			}
			text.WriteString(p.Text)
		}

		if call == nil {
			if text.Len() == 0 {
				return "", fmt.Errorf("empty response from model")
			}
			return text.String(), nil
		}

		result := runTool(call, state)
		contents = append(contents,
			geminiContent{Role: "model", Parts: []geminiPart{{FunctionCall: call, ThoughtSignature: signature}}},
			geminiContent{Role: "user", Parts: []geminiPart{{
				FunctionResponse: &functionResponsePart{Name: call.Name, Response: result},
			}}},
		)
	}

	return "", fmt.Errorf("copilot could not produce a final answer after tool calls")
}

func runTool(call *functionCall, state *AppState) map[string]interface{} {
	switch call.Name {
	case "search_flights":
		originCountry, _ := call.Args["originCountry"].(string)
		callsign, _ := call.Args["callsignContains"].(string)
		limit := 10
		if l, ok := call.Args["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}
		if limit > 25 {
			limit = 25
		}
		matches := state.SearchFlights(originCountry, callsign, limit)
		return map[string]interface{}{
			"matchCount": len(matches),
			"flights":    matches,
		}
	case "get_high_risk_flights":
		minScore := 50
		if m, ok := call.Args["minScore"].(float64); ok && m > 0 {
			minScore = int(m)
		}
		limit := 10
		if l, ok := call.Args["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}
		matches := state.HighRiskFlights(minScore, limit)
		return map[string]interface{}{
			"matchCount": len(matches),
			"flights":    matches,
		}
	case "get_chokepoint_congestion":
		return map[string]interface{}{
			"chokepoints": state.ChokepointCongestion(),
		}
	default:
		return map[string]interface{}{"error": "unknown tool: " + call.Name}
	}
}

func callGemini(apiKey, system string, contents []geminiContent, tools []geminiTool) ([]geminiPart, error) {
	reqBody := geminiRequest{
		SystemInstruction: geminiContent{Parts: []geminiPart{{Text: system}}},
		Contents:          contents,
		Tools:             tools,
		GenerationConfig:  geminiGenConfig{MaxOutputTokens: 500},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, geminiURL(apiKey), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed geminiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("gemini error: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 {
		return nil, fmt.Errorf("empty response from model (status %d): %s", resp.StatusCode, string(body))
	}
	return parsed.Candidates[0].Content.Parts, nil
}
