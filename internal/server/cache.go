package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CachedData struct {
	FetchedAt time.Time `json:"fetched_at"`
	ESMCodes  []string  `json:"esm_codes"`
	Nodes     []ESMNode `json:"nodes"`
}

func LoadCache(cachePath string) (*CachedData, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cached CachedData
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

func SaveCache(cachePath string, data *CachedData) error {
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, jsonData, 0600)
}

// FetchServers fetches servers for a single ESM code.
func FetchServers(baseURL, esmCode string) ([]ESMNode, error) {
	url := fmt.Sprintf("%s/%s/servers?suincluded=true", baseURL, esmCode)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Header.IsSuccessful {
		return nil, fmt.Errorf("API error: %s", apiResp.Header.ResultMessage)
	}

	return apiResp.Body.Data, nil
}

// FetchAllServers fetches servers for multiple ESM codes concurrently.
func FetchAllServers(baseURL string, esmCodes []string) ([]ESMNode, error) {
	if len(esmCodes) == 0 {
		return nil, fmt.Errorf("no ESM codes configured")
	}

	type result struct {
		nodes []ESMNode
		err   error
		code  string
	}

	var wg sync.WaitGroup
	results := make(chan result, len(esmCodes))

	for _, code := range esmCodes {
		wg.Add(1)
		go func(esmCode string) {
			defer wg.Done()
			nodes, err := FetchServers(baseURL, esmCode)
			results <- result{nodes: nodes, err: err, code: esmCode}
		}(code)
	}

	wg.Wait()
	close(results)

	var allNodes []ESMNode
	var errs []string
	for r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", r.code, r.err))
			continue
		}
		allNodes = append(allNodes, r.nodes...)
	}

	if len(errs) > 0 && len(allNodes) == 0 {
		return nil, fmt.Errorf("all API requests failed:\n  %s", strings.Join(errs, "\n  "))
	}

	return allNodes, nil
}
