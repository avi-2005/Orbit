package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"sync"
)

const historyFile = "orbit_history.jsonl"

// History is an append-only event log — every insight ever detected gets
// written as one JSON line. This is the simplest form of the "event
// sourcing" pattern: instead of a database, the log itself is the record,
// and we replay it on startup to rebuild lifetime counters. It survives
// restarts without adding a database driver/toolchain dependency, which
// matters a lot for a project meant to be easy to clone and run.
type History struct {
	mu     sync.Mutex
	file   *os.File
	byType map[string]int
	total  int
}

func OpenHistory() *History {
	h := &History{byType: make(map[string]int)}

	// Replay existing log (if any) to rebuild lifetime counts.
	if existing, err := os.Open(historyFile); err == nil {
		scanner := bufio.NewScanner(existing)
		for scanner.Scan() {
			var ins Insight
			if json.Unmarshal(scanner.Bytes(), &ins) == nil {
				h.byType[ins.Type]++
				h.total++
			}
		}
		existing.Close()
	}

	f, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("WARNING: could not open history log, insights won't persist across restarts:", err)
		return h
	}
	h.file = f
	log.Printf("history log loaded: %d insights from previous sessions\n", h.total)
	return h
}

func (h *History) Record(insights []Insight) {
	if len(insights) == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, ins := range insights {
		h.byType[ins.Type]++
		h.total++

		if h.file == nil {
			continue
		}
		line, err := json.Marshal(ins)
		if err != nil {
			continue
		}
		h.file.Write(line)
		h.file.Write([]byte("\n"))
	}
}

func (h *History) Stats() map[string]interface{} {
	h.mu.Lock()
	defer h.mu.Unlock()

	byType := make(map[string]int, len(h.byType))
	for k, v := range h.byType {
		byType[k] = v
	}
	return map[string]interface{}{
		"totalInsightsAllTime": h.total,
		"byType":               byType,
	}
}
