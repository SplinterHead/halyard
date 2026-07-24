package manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// GatherFromAgents makes a concurrent HTTP GET request to each agent's endpoint,
// unmarshals the JSON response into a slice of T, applies the mutator function (if provided)
// to set node-specific data on each item, and returns the aggregated results.
func GatherFromAgents[T any](client *http.Client, endpointPath string, agents []AgentRecord, mutator func(item *T, nodeID string)) []T {
	var wg sync.WaitGroup
	var mu sync.Mutex
	allResults := make([]T, 0)

	for _, agent := range agents {
		wg.Add(1)
		go func(taskIP, nodeID string) {
			defer wg.Done()
			url := fmt.Sprintf("https://%s:9090%s", taskIP, endpointPath)
			resp, err := client.Get(url)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			var items []T
			if err := json.NewDecoder(resp.Body).Decode(&items); err == nil {
				mu.Lock()
				for i := range items {
					if mutator != nil {
						mutator(&items[i], nodeID)
					}
					allResults = append(allResults, items[i])
				}
				mu.Unlock()
			}
		}(agent.IP, agent.NodeID)
	}

	wg.Wait()
	return allResults
}
