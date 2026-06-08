package agentclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ParseError checks the response status code against expected successful status codes.
// If the status is unexpected, it reads the error message from the response body and returns a formatted error.
func ParseError(resp *http.Response, expectedStatuses ...int) error {
	isExpected := false
	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			isExpected = true
			break
		}
	}
	if isExpected {
		return nil
	}

	// Try to decode JSON error first
	var temp struct {
		Error string `json:"error"`
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err == nil && len(bodyBytes) > 0 {
		if json.Unmarshal(bodyBytes, &temp) == nil && temp.Error != "" {
			return fmt.Errorf("agent error: %s", temp.Error)
		}
		// Fallback to reading raw body if it is not JSON
		rawMsg := string(bodyBytes)
		if len(rawMsg) > 256 {
			rawMsg = rawMsg[:256] + "..."
		}
		return fmt.Errorf("agent error: %s (status %d)", rawMsg, resp.StatusCode)
	}

	return fmt.Errorf("agent returned status %d", resp.StatusCode)
}
