package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RCASession struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	ClusterName string `json:"clusterName"`
}

type RCAResponse struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}

type Evidence struct {
	Query   string `json:"query"`
	Snippet string `json:"snippet"`
}

type RCAPollResponse struct {
	SessionID          string                 `json:"sessionId"`
	IsComplete         bool                   `json:"isComplete"`
	IsFailed           bool                   `json:"isFailed"`
	IsStuck            bool                   `json:"isStuck"`
	ProblemShort       string                 `json:"problemShort"`
	Recommendation     string                 `json:"recommendation"`
	WhatHappened       []string               `json:"whatHappened"`
	EvidenceCollection []Evidence             `json:"evidenceCollection"`
	Operations         []string               `json:"operations"`
	RawData            map[string]interface{} `json:"-"`
}

type KomodorCluster struct {
	APIServerURL string            `json:"apiServerUrl"`
	ClusterID    string            `json:"clusterId"`
	Name         string            `json:"name"`
	Tags         map[string]string `json:"tags"`
}

type KomodorClustersResponse struct {
	Data struct {
		Clusters []KomodorCluster `json:"clusters"`
	} `json:"data"`
}

func triggerRCA(config *Config) (*RCAResponse, error) {
	session := &RCASession{
		Namespace:   config.Namespace,
		Name:        config.Name,
		Kind:        config.Kind,
		ClusterName: config.KomodorClusterName,
	}

	jsonData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	url := fmt.Sprintf("%s/api/v2/klaudia/rca/sessions", config.KomodorBaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.KomodorAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("RCA failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var rcaResp RCAResponse
	if err := json.Unmarshal(body, &rcaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &rcaResp, nil
}

func pollRCAResults(config *Config, sessionID string) error {
	config.TUI.DisplayMessage("\n🔄 Starting live RCA monitoring...")
	config.TUI.DisplayMessage("Press Ctrl+C to stop monitoring")
	config.TUI.DisplayMessage("")

	var lastDisplayedData string
	pollCount := 0
	maxRetries := 72
	retryCount := 0

	for {
		pollCount++
		url := fmt.Sprintf("%s/api/v2/klaudia/rca/sessions/%s", config.KomodorBaseURL, sessionID)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			retryCount++
			config.TUI.DisplayProgressIndicator(fmt.Sprintf("❌ Poll failed: %v (retry %d/%d)", err, retryCount, maxRetries))
			if retryCount >= maxRetries {
				config.TUI.DisplayError(fmt.Sprintf("Failed to create poll request after %d retries", maxRetries), err)
				return fmt.Errorf("failed to create poll request after %d retries: %w", maxRetries, err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		req.Header.Set("x-api-key", config.KomodorAPIKey)

		client := &http.Client{Timeout: 360 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			retryCount++
			config.TUI.DisplayProgressIndicator(fmt.Sprintf("❌ Poll failed: %v (retry %d/%d)", err, retryCount, maxRetries))
			if retryCount >= maxRetries {
				config.TUI.DisplayError(fmt.Sprintf("Failed to poll session after %d retries", maxRetries), err)
				return fmt.Errorf("failed to poll session after %d retries: %w", maxRetries, err)
			}
			time.Sleep(5 * time.Second)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			retryCount++
			config.TUI.DisplayProgressIndicator(fmt.Sprintf("❌ Failed to read response: %v (retry %d/%d)", err, retryCount, maxRetries))
			if retryCount >= maxRetries {
				config.TUI.DisplayError(fmt.Sprintf("Failed to read response after %d retries", maxRetries), err)
				return fmt.Errorf("failed to read response after %d retries: %w", maxRetries, err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			retryCount++
			config.TUI.DisplayProgressIndicator(fmt.Sprintf("❌ Polling failed (HTTP %d): %s (retry %d/%d)", resp.StatusCode, string(body), retryCount, maxRetries))
			if retryCount >= maxRetries {
				config.TUI.DisplayError(fmt.Sprintf("Polling failed after %d retries (HTTP %d)", maxRetries, resp.StatusCode), fmt.Errorf("%s", string(body)))
				return fmt.Errorf("polling failed after %d retries (HTTP %d): %s", maxRetries, resp.StatusCode, string(body))
			}
			time.Sleep(5 * time.Second)
			continue
		}

		retryCount = 0

		var rawData map[string]interface{}
		if err := json.Unmarshal(body, &rawData); err != nil {
			retryCount++
			config.TUI.DisplayProgressIndicator(fmt.Sprintf("❌ Failed to parse raw response: %v (retry %d/%d)", err, retryCount, maxRetries))
			logMessage("Raw response: %s", string(body))
			if retryCount >= maxRetries {
				config.TUI.DisplayError(fmt.Sprintf("Failed to parse raw response after %d retries", maxRetries), err)
				return fmt.Errorf("failed to parse raw response after %d retries: %w", maxRetries, err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		var pollResp RCAPollResponse
		if err := json.Unmarshal(body, &pollResp); err != nil {
			retryCount++
			config.TUI.DisplayProgressIndicator(fmt.Sprintf("❌ Failed to parse structured response: %v (retry %d/%d)", err, retryCount, maxRetries))
			logMessage("Response body: %s", string(body))
			if retryCount >= maxRetries {
				config.TUI.DisplayError(fmt.Sprintf("Failed to parse structured response after %d retries", maxRetries), err)
				return fmt.Errorf("failed to parse structured response after %d retries: %w", maxRetries, err)
			}
			time.Sleep(5 * time.Second)
			continue
		}
		pollResp.RawData = rawData

		retryCount = 0

		currentData := fmt.Sprintf("%s|%s|%s|%d|%d|%d",
			pollResp.ProblemShort,
			pollResp.Recommendation,
			pollResp.SessionID,
			len(pollResp.WhatHappened),
			len(pollResp.EvidenceCollection),
			len(pollResp.Operations))

		if currentData != lastDisplayedData {
			logMessage("RCA data updated - refreshing display")
			config.TUI.ClearScreen()
			config.TUI.DisplayLiveRCAResults(&pollResp, pollCount)
			lastDisplayedData = currentData
		} else {
			config.TUI.DisplayProgressIndicator("⏳ In Progress...")
		}

		if pollResp.IsComplete {
			config.TUI.ClearScreen()
			config.TUI.DisplayFinalRCAResults(&pollResp)
			logMessage("RCA completed successfully.")
			break
		}

		if pollCount > 300 {
			config.TUI.DisplayMessage("\n⏰ Timeout reached (15 minutes). RCA may still be processing.")
			logMessage("RCA polling timed out after %d attempts", pollCount)
			break
		}

		time.Sleep(2 * time.Second)
	}

	config.TUI.WaitForExit()
	return nil
}

func fetchRCAStatus(config *Config, sessionID string) (*RCAPollResponse, error) {
	url := fmt.Sprintf("%s/api/v2/klaudia/rca/sessions/%s", config.KomodorBaseURL, sessionID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", config.KomodorAPIKey)

	client := &http.Client{Timeout: 360 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var rawData map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw response: %w", err)
	}

	var pollResp RCAPollResponse
	if err := json.Unmarshal(body, &pollResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	pollResp.RawData = rawData

	return &pollResp, nil
}

func fetchKomodorClusters(apiKey, baseURL string) ([]KomodorCluster, error) {
	logMessage("Fetching Komodor clusters from API...")
	url := fmt.Sprintf("%s/api/v2/clusters", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logMessage("ERROR: Failed to create API request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logMessage("ERROR: Failed to make API request: %v", err)
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logMessage("ERROR: Failed to read API response: %v", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		logMessage("ERROR: API request failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var clustersResp KomodorClustersResponse
	if err := json.Unmarshal(body, &clustersResp); err != nil {
		logMessage("ERROR: Failed to parse API response: %v", err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logMessage("Successfully fetched %d clusters from Komodor API", len(clustersResp.Data.Clusters))
	return clustersResp.Data.Clusters, nil
}
