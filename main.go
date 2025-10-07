package main

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

type Config struct {
	KomodorAPIKey      string
	LocalClusterName   string
	KomodorClusterName string
	KomodorBaseURL     string
	Namespace          string
	Name               string
	Kind               string
	Context            string
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			logMessage("FATAL: Application crashed with panic: %v", r)
			fmt.Fprintf(os.Stderr, "FATAL: Application crashed: %v\n", r)
			os.Exit(1)
		}
	}()

	loadEnvironmentFiles()

	rootCmd := &cobra.Command{
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
	rootCmd.Flags().String("context", "", "Kubernetes context name")
	rootCmd.Flags().String("base-url", "https://api.komodor.com", "Komodor API base URL")
	rootCmd.Flags().Bool("poll", false, "Poll for RCA completion")
	rootCmd.Flags().Bool("background", false, "Run in background mode")

	if err := rootCmd.Execute(); err != nil {
		logMessage("FATAL: Command execution failed: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func loadEnvironmentFiles() {
	if err := godotenv.Load(); err != nil {
		if err := godotenv.Load(".env"); err != nil {
			if homeDir, err := os.UserHomeDir(); err == nil {
				godotenv.Load(homeDir + "/.k9s-komodor-rca/.env")
			}
		}
	}
}

func runRCA(cmd *cobra.Command, args []string) error {
	config, err := loadConfig(cmd)
	if err != nil {
		logMessage("FATAL: Configuration error: %v", err)
		displayError("Configuration error", err)
		return err
	}

	if err := validateConfig(config); err != nil {
		logMessage("FATAL: Validation error: %v", err)
		displayError("Validation error", err)
		return err
	}

	logMessage("Config: APIKey=%s, Cluster=%s, BaseURL=%s, Namespace=%s, Name=%s, Kind=%s, Context=%s",
		maskAPIKey(config.KomodorAPIKey), config.KomodorClusterName, config.KomodorBaseURL,
		config.Namespace, config.Name, config.Kind, config.Context)

	logMessage("🚀 Triggering RCA for %s: %s in namespace: %s on cluster: %s",
		config.Kind, config.Name, config.Namespace, config.KomodorClusterName)

	session, err := triggerRCA(config)
	if err != nil {
		logMessage("FATAL: RCA trigger failed: %v", err)
		displayError("RCA trigger failed", err)
		return fmt.Errorf("failed to trigger RCA: %w", err)
	}

	if session.SessionID == "" {
		logMessage("FATAL: No session ID received from Komodor API")
		displayError("No session ID received from Komodor API", fmt.Errorf("empty session ID"))
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
	config := &Config{
		KomodorAPIKey:    getEnvOrFlag(cmd, "KOMODOR_API_KEY", "api-key"),
		KomodorBaseURL:   getEnvOrFlag(cmd, "KOMODOR_BASE_URL", "base-url"),
		Namespace:        getEnvOrFlag(cmd, "NAMESPACE", "namespace"),
		Name:             getEnvOrFlag(cmd, "NAME", "name"),
		Kind:             getEnvOrFlag(cmd, "KIND", "kind"),
		Context:          getEnvOrFlag(cmd, "CONTEXT", "context"),
		LocalClusterName: getEnvOrFlag(cmd, "CLUSTER", "cluster"),
	}

	if config.KomodorBaseURL == "" {
		config.KomodorBaseURL = "https://api.komodor.com"
	}

	if config.Context != "" && config.Context != config.LocalClusterName {
		config.LocalClusterName = config.Context
	}

	if config.LocalClusterName == "" {
		logMessage("ERROR: No cluster provided")
		return nil, fmt.Errorf("cluster is required (use --cluster flag or CLUSTER env var)")
	}

	komodorCluster, err := resolveKomodorCluster(config.KomodorAPIKey, config.KomodorBaseURL, config.LocalClusterName)
	if err != nil {
		logMessage("ERROR: Failed to resolve Komodor cluster: %v", err)
		return nil, err
	}
	config.KomodorClusterName = komodorCluster
	logMessage("Local cluster: %s, Komodor cluster: %s", config.LocalClusterName, config.KomodorClusterName)
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

func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "(empty)"
	}
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}

func logMessage(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	logFile := os.Getenv("HOME") + "/.k9s-komodor-rca/k9s_komodor_logs.txt"
	if f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		defer f.Close()
		if _, err := f.WriteString(logEntry); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write to log file: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
	}
}
