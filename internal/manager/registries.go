package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/SplinterHead/halyard/api"
)

type RegistryManager struct {
	db *DB
}

func NewRegistryManager(db *DB) *RegistryManager {
	return &RegistryManager{db: db}
}

func (m *RegistryManager) ListRegistries() ([]api.Registry, error) {
	rows, err := m.db.Query("SELECT id, name, url, username, password, created_at FROM registries ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	registries := make([]api.Registry, 0)
	for rows.Next() {
		var r api.Registry
		if err := rows.Scan(&r.ID, &r.Name, &r.URL, &r.Username, &r.Password, &r.CreatedAt); err != nil {
			return nil, err
		}
		registries = append(registries, r)
	}
	return registries, nil
}

func (m *RegistryManager) AddRegistry(r api.Registry) (api.Registry, error) {
	r.ID = uuid.New().String()
	r.CreatedAt = time.Now()

	_, err := m.db.Exec(
		"INSERT INTO registries (id, name, url, username, password, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		r.ID, r.Name, r.URL, r.Username, r.Password, r.CreatedAt,
	)
	if err != nil {
		return api.Registry{}, err
	}
	return r, nil
}

func (m *RegistryManager) UpdateRegistry(r api.Registry) (api.Registry, error) {
	_, err := m.db.Exec(
		"UPDATE registries SET name = ?, url = ?, username = ?, password = ? WHERE id = ?",
		r.Name, r.URL, r.Username, r.Password, r.ID,
	)
	if err != nil {
		return api.Registry{}, err
	}
	return r, nil
}

func (m *RegistryManager) DeleteRegistry(id string) error {
	_, err := m.db.Exec("DELETE FROM registries WHERE id = ?", id)
	return err
}

// FetchDockerHubRateLimit queries Docker Hub for the current pull limits and remaining quota
func FetchDockerHubRateLimit(ctx context.Context, username, password string) (*api.DockerHubRateLimit, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Get registry auth token
	tokenURL := "https://auth.docker.io/token?service=registry.docker.io&scope=repository:ratelimitpreview/test:pull"
	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return nil, err
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth token request failed: %s", resp.Status)
	}

	var authResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	// 2. Head manifest to obtain rate limit headers
	manifestURL := "https://registry-1.docker.io/v2/ratelimitpreview/test/manifests/latest"
	req, err = http.NewRequestWithContext(ctx, "HEAD", manifestURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+authResp.Token)
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limitVal := resp.Header.Get("RateLimit-Limit")
	remVal := resp.Header.Get("RateLimit-Remaining")
	resetVal := resp.Header.Get("RateLimit-Reset")

	if limitVal == "" || remVal == "" {
		return nil, fmt.Errorf("rate limit headers missing")
	}

	parseHeader := func(h string) int {
		parts := strings.Split(h, ";")
		val, _ := strconv.Atoi(parts[0])
		return val
	}

	limit := parseHeader(limitVal)
	remaining := parseHeader(remVal)
	reset := 0
	if resetVal != "" {
		reset, _ = strconv.Atoi(resetVal)
	}

	return &api.DockerHubRateLimit{
		Limit:     limit,
		Remaining: remaining,
		Reset:     reset,
	}, nil
}

// GetEncodedRegistryAuth returns base64 url-encoded credentials matching the given repository
func GetEncodedRegistryAuth(repo string, registries []api.Registry) string {
	registryHost := "docker.io" // Default Docker Hub
	parts := strings.Split(repo, "/")
	if len(parts) > 1 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		registryHost = parts[0]
	}

	var matchedUser, matchedPass, matchedURL string
	
	cleanRegistryURL := func(u string) string {
		u = strings.TrimPrefix(u, "https://")
		u = strings.TrimPrefix(u, "http://")
		u = strings.TrimSuffix(u, "/v1")
		u = strings.TrimSuffix(u, "/v1/")
		u = strings.TrimSuffix(u, "/v2")
		u = strings.TrimSuffix(u, "/v2/")
		u = strings.TrimSuffix(u, "/")
		return strings.ToLower(u)
	}

	cleanHost := cleanRegistryURL(registryHost)

	for _, reg := range registries {
		cleanURL := cleanRegistryURL(reg.URL)
		if strings.Contains(cleanURL, cleanHost) || strings.Contains(cleanHost, cleanURL) {
			matchedUser = reg.Username
			matchedPass = reg.Password
			matchedURL = reg.URL
			break
		}
	}

	// If it's docker.io / Docker Hub, and we found no matched credential, look for named Docker Hub registries
	if registryHost == "docker.io" && matchedUser == "" {
		for _, reg := range registries {
			cleanURL := cleanRegistryURL(reg.URL)
			if strings.Contains(cleanURL, "docker.io") || strings.Contains(strings.ToLower(reg.Name), "docker hub") {
				matchedUser = reg.Username
				matchedPass = reg.Password
				matchedURL = reg.URL
				break
			}
		}
	}

	if matchedUser == "" || matchedPass == "" {
		return ""
	}

	authConfig := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		ServerAddress string `json:"serveraddress,omitempty"`
	}{
		Username:      matchedUser,
		Password:      matchedPass,
		ServerAddress: matchedURL,
	}

	jsonData, err := json.Marshal(authConfig)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(jsonData)
}
