package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var version = "dev"

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

type RCAPollResponse struct {
	SessionID       string                 `json:"sessionId"`
	IsComplete      bool                   `json:"isComplete"`
	ProblemShort    string                 `json:"problemShort"`
	Recommendation  string                 `json:"recommendation"`
	WhatHappened    []string               `json:"whatHappened"`
	EvidenceQueries []string               `json:"evidenceQueries"`
	Operations      []string               `json:"operations"`
	RawData         map[string]interface{} `json:"-"`
}

type ClusterMapping struct {
	Mapping map[string]string `yaml:"mapping"`
}

type Config struct {
	KomodorAPIKey      string
	KomodorClusterName string
	KomodorBaseURL     string
	Namespace          string
	Name               string
	Kind               string
}

func main() {
	if err := godotenv.Load(); err != nil {
		if err := godotenv.Load(".env"); err != nil {
			if homeDir, err := os.UserHomeDir(); err == nil {
				if err := godotenv.Load(homeDir + "/.k9s-komodor-rca/.env"); err != nil {

				}
			}
		}
	}

	var rootCmd = &cobra.Command{
		Use:   "k9s-rca",
		Short: "K9s Komodor RCA Plugin",
		Long:  "A Go-based plugin for triggering Komodor Root Cause Analysis from K9s",
		RunE:  runRCA,
	}

	rootCmd.Flags().String("kind", "", "Kubernetes resource kind (Pod, Deployment, Service, etc.)")
	rootCmd.Flags().String("namespace", "", "Kubernetes namespace")
	rootCmd.Flags().String("name", "", "Kubernetes resource name")
	rootCmd.Flags().String("api-key", "", "Komodor API key")
	rootCmd.Flags().String("cluster", "", "Kubernetes cluster name")
	rootCmd.Flags().String("base-url", "https://api.komodor.com", "Komodor API base URL")
	rootCmd.Flags().Bool("poll", false, "Poll for RCA completion")
	rootCmd.Flags().Bool("background", false, "Run in background mode")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runRCA(cmd *cobra.Command, args []string) error {
	config, err := loadConfig(cmd)
	if err != nil {
		fmt.Printf("\n❌ Configuration error: %v\n", err)
		fmt.Println("\n📋 Press Enter to exit...")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		return err
	}

	if err := validateConfig(config); err != nil {
		fmt.Printf("\n❌ Validation error: %v\n", err)
		fmt.Println("\n📋 Press Enter to exit...")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		return err
	}

	logMessage("🚀 Triggering RCA for %s: %s in namespace: %s on cluster: %s", config.Kind, config.Name, config.Namespace, config.KomodorClusterName)

	session, err := triggerRCA(config)
	if err != nil {
		fmt.Printf("\n❌ RCA trigger failed: %v\n", err)
		fmt.Println("\n📋 Press Enter to exit...")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		return fmt.Errorf("failed to trigger RCA: %w", err)
	}

	if session.SessionID == "" {
		fmt.Printf("\n❌ No session ID received from Komodor API\n")
		fmt.Println("\n📋 Press Enter to exit...")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		return fmt.Errorf("no session ID received from Komodor API")
	}

	logMessage("\n✅ RCA triggered successfully! Session ID: %s", session.SessionID)

	shouldPoll, _ := cmd.Flags().GetBool("poll")
	isBackground, _ := cmd.Flags().GetBool("background")

	if shouldPoll || !isBackground {
		fmt.Println("\n🔄 Starting RCA monitoring...")
		return pollRCAResults(config, session.SessionID)
	}

	return nil
}

func loadConfig(cmd *cobra.Command) (*Config, error) {
	config := &Config{}

	config.KomodorAPIKey = getEnvOrFlag(cmd, "KOMODOR_API_KEY", "api-key")
	config.KomodorClusterName = getEnvOrFlag(cmd, "KOMODOR_CLUSTER_NAME", "cluster")
	config.KomodorBaseURL = getEnvOrFlag(cmd, "KOMODOR_BASE_URL", "base-url")
	config.Namespace = getEnvOrFlag(cmd, "NAMESPACE", "namespace")
	config.Name = getEnvOrFlag(cmd, "NAME", "name")
	config.Kind = getEnvOrFlag(cmd, "KIND", "kind")

	if config.KomodorBaseURL == "" {
		config.KomodorBaseURL = "https://api.komodor.com"
	}

	clusterMapping, err := loadClusterMapping()
	if err == nil {
		config.KomodorClusterName = convertClusterName(config.KomodorClusterName, clusterMapping)
	}

	return config, nil
}

func getEnvOrFlag(cmd *cobra.Command, envVar, flagName string) string {
	if value, _ := cmd.Flags().GetString(flagName); value != "" {
		return value
	}
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return ""
}

func validateConfig(config *Config) error {
	if config.KomodorAPIKey == "" {
		return fmt.Errorf("KOMODOR_API_KEY environment variable is required")
	}
	if config.KomodorClusterName == "" {
		return fmt.Errorf("KOMODOR_CLUSTER_NAME environment variable is required")
	}
	if config.Namespace == "" {
		return fmt.Errorf("namespace is required (use --namespace flag)")
	}
	if config.Name == "" {
		return fmt.Errorf("name is required (use --name flag)")
	}
	if config.Kind == "" {
		return fmt.Errorf("kind is required (use --kind flag)")
	}
	return nil
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

	// logMessage("HTTP %d - %s", resp.StatusCode, string(body))

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
	fmt.Println("\n🔄 Starting live RCA monitoring...")
	fmt.Println("Press Ctrl+C to stop monitoring")
	fmt.Println()

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
			fmt.Printf("\r❌ Poll failed: %v (retry %d/%d)", err, retryCount, maxRetries)
			if retryCount >= maxRetries {
				fmt.Printf("\n❌ Failed to create poll request after %d retries: %v\n", maxRetries, err)
				fmt.Println("\n📋 Press Enter to exit...")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
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
			fmt.Printf("\r❌ Poll failed: %v (retry %d/%d)", err, retryCount, maxRetries)
			if retryCount >= maxRetries {
				fmt.Printf("\n❌ Failed to poll session after %d retries: %v\n", maxRetries, err)
				fmt.Println("\n📋 Press Enter to exit...")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				return fmt.Errorf("failed to poll session after %d retries: %w", maxRetries, err)
			}
			time.Sleep(5 * time.Second)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			retryCount++
			fmt.Printf("\r❌ Failed to read response: %v (retry %d/%d)", err, retryCount, maxRetries)
			if retryCount >= maxRetries {
				fmt.Printf("\n❌ Failed to read response after %d retries: %v\n", maxRetries, err)
				fmt.Println("\n📋 Press Enter to exit...")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				return fmt.Errorf("failed to read response after %d retries: %w", maxRetries, err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			retryCount++
			fmt.Printf("\r❌ Polling failed (HTTP %d): %s (retry %d/%d)", resp.StatusCode, string(body), retryCount, maxRetries)
			if retryCount >= maxRetries {
				fmt.Printf("\n❌ Polling failed after %d retries (HTTP %d): %s\n", maxRetries, resp.StatusCode, string(body))
				fmt.Println("\n📋 Press Enter to exit...")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				return fmt.Errorf("polling failed after %d retries (HTTP %d): %s", maxRetries, resp.StatusCode, string(body))
			}
			time.Sleep(5 * time.Second)
			continue
		}

		retryCount = 0

		var rawData map[string]interface{}
		if err := json.Unmarshal(body, &rawData); err != nil {
			retryCount++
			fmt.Printf("\r❌ Failed to parse raw response: %v (retry %d/%d)", err, retryCount, maxRetries)
			logMessage("Raw response: %s", string(body))
			if retryCount >= maxRetries {
				fmt.Printf("\n❌ Failed to parse raw response after %d retries: %v\n", maxRetries, err)
				fmt.Println("\n📋 Press Enter to exit...")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				return fmt.Errorf("failed to parse raw response after %d retries: %w", maxRetries, err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		var pollResp RCAPollResponse
		if err := json.Unmarshal(body, &pollResp); err != nil {
			retryCount++
			fmt.Printf("\r❌ Failed to parse structured response: %v (retry %d/%d)", err, retryCount, maxRetries)
			logMessage("Response body: %s", string(body))
			if retryCount >= maxRetries {
				fmt.Printf("\n❌ Failed to parse structured response after %d retries: %v\n", maxRetries, err)
				fmt.Println("\n📋 Press Enter to exit...")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
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
			len(pollResp.EvidenceQueries),
			len(pollResp.Operations))

		if currentData != lastDisplayedData {
			logMessage("RCA data updated - refreshing display")
			clearScreen()
			displayLiveRCAResults(&pollResp, pollCount)
			lastDisplayedData = currentData
		} else {
			fmt.Printf("\r⏳ In Progress...")
		}

		if pollResp.IsComplete {
			clearScreen()
			displayFinalRCAResults(&pollResp)
			logMessage("RCA completed successfully.")
			break
		}

		if pollCount > 300 {
			fmt.Println("\n⏰ Timeout reached (15 minutes). RCA may still be processing.")
			logMessage("RCA polling timed out after %d attempts", pollCount)
			break
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Println("\n📋 Press Enter to exit...")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	return nil
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func displayLiveRCAResults(results *RCAPollResponse, pollCount int) {
	fmt.Println("🔍 LIVE RCA ANALYSIS")
	fmt.Println("====================")
	fmt.Printf("📊 Poll Count: %d | Last Update: %s\n", pollCount, time.Now().Format("15:04:05"))
	fmt.Printf("🆔 Session ID: %s\n", results.SessionID)
	fmt.Printf("✅ Status: %s\n", getStatusText(results.IsComplete))
	fmt.Println()

	if results.ProblemShort != "" {
		fmt.Printf("📋 Problem: %s\n", results.ProblemShort)
	}

	if results.Recommendation != "" {
		fmt.Printf("💡 Recommendation: %s\n", results.Recommendation)
	}

	fmt.Println()
	fmt.Println("📝 What Happened:")
	if len(results.WhatHappened) > 0 {
		for i, event := range results.WhatHappened {
			fmt.Printf("  %d. %s\n", i+1, event)
		}
	} else {
		fmt.Println("  ⏳ Waiting for data...")
	}

	fmt.Println()
	fmt.Println("🔍 Evidence:")
	if len(results.EvidenceQueries) > 0 {
		for i, query := range results.EvidenceQueries {
			fmt.Printf("  %d. %s\n", i+1, query)
		}
	} else {
		fmt.Println("  ⏳ Waiting for data...")
	}

	fmt.Println()
	fmt.Println("📊 Operations:")
	if len(results.Operations) > 0 {
		for i, operation := range results.Operations {
			fmt.Printf("  %d. %s\n", i+1, operation)
		}
	} else {
		fmt.Println("  ⏳ Waiting for data...")
	}

	fmt.Println()
	fmt.Println("====================")
	fmt.Println("Press Ctrl+C to stop monitoring")
}

func displayFinalRCAResults(results *RCAPollResponse) {
	fmt.Println("✅ RCA ANALYSIS COMPLETED!")
	fmt.Println("==========================")
	fmt.Printf("🆔 Session ID: %s\n", results.SessionID)
	fmt.Printf("⏰ Completed at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()

	if results.ProblemShort != "" {
		fmt.Printf("📋 Problem: %s\n", results.ProblemShort)
	}

	if results.Recommendation != "" {
		fmt.Printf("💡 Recommendation: %s\n", results.Recommendation)
	}

	fmt.Println()
	fmt.Println("📝 What Happened:")
	if len(results.WhatHappened) > 0 {
		for i, event := range results.WhatHappened {
			fmt.Printf("  %d. %s\n", i+1, event)
		}
	} else {
		fmt.Println("  • No what happened data available")
	}

	fmt.Println()
	fmt.Println("🔍 Evidence:")
	if len(results.EvidenceQueries) > 0 {
		for i, query := range results.EvidenceQueries {
			fmt.Printf("  %d. %s\n", i+1, query)
		}
	} else {
		fmt.Println("  • No evidence queries found")
	}

	fmt.Println()
	fmt.Println("📊 Operations Performed:")
	if len(results.Operations) > 0 {
		for i, operation := range results.Operations {
			fmt.Printf("  %d. %s\n", i+1, operation)
		}
	} else {
		fmt.Println("  • No operations data available")
	}

	fmt.Println()
	fmt.Println("==========================")
}

func displayDynamicFields(results *RCAPollResponse) {
	if results.RawData != nil {
		fmt.Println()
		fmt.Println("🔧 Additional Fields:")
		for key, value := range results.RawData {
			if !isKnownField(key) {
				fmt.Printf("  • %s: %v\n", key, value)
			}
		}
	}
}

func isKnownField(fieldName string) bool {
	knownFields := []string{
		"sessionId", "isComplete", "problemShort", "recommendation",
		"whatHappened", "evidenceQueries", "operations",
	}
	for _, field := range knownFields {
		if strings.EqualFold(fieldName, field) {
			return true
		}
	}
	return false
}

func getStatusText(isComplete bool) string {
	if isComplete {
		return "✅ Complete"
	}
	return "⏳ In Progress"
}

func logMessage(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	fmt.Println(message)

	logFile := os.Getenv("HOME") + "/.k9s_komodor_logs.txt"
	if f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		defer f.Close()
		if _, err := f.WriteString(logEntry); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write to log file: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
	}
}

func loadClusterMapping() (*ClusterMapping, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	clusterMappingFile := homeDir + "/.k9s-komodor-rca/clusters.yaml"

	data, err := os.ReadFile(clusterMappingFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &ClusterMapping{Mapping: make(map[string]string)}, nil
		}
		return nil, fmt.Errorf("failed to read cluster mapping file: %w", err)
	}

	var mapping ClusterMapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("failed to parse cluster mapping file: %w", err)
	}

	if mapping.Mapping == nil {
		mapping.Mapping = make(map[string]string)
	}

	return &mapping, nil
}

func convertClusterName(localClusterName string, mapping *ClusterMapping) string {
	if mapping == nil || mapping.Mapping == nil {
		return localClusterName
	}

	if komodorClusterName, exists := mapping.Mapping[localClusterName]; exists {
		return komodorClusterName
	}

	return localClusterName
}
