package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

type ClusterMapping struct {
	Mapping map[string]string `yaml:"mapping"`
}

func loadClusterMapping() (*ClusterMapping, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &ClusterMapping{Mapping: make(map[string]string)}, nil
	}

	clusterMappingFile := homeDir + "/.k9s-komodor-rca/clusters.yaml"

	data, err := os.ReadFile(clusterMappingFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &ClusterMapping{Mapping: make(map[string]string)}, nil
		}
		return &ClusterMapping{Mapping: make(map[string]string)}, nil
	}

	var mapping ClusterMapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return &ClusterMapping{Mapping: make(map[string]string)}, nil
	}

	if mapping.Mapping == nil {
		mapping.Mapping = make(map[string]string)
	}

	return &mapping, nil
}

func saveClusterMapping(mapping *ClusterMapping) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := homeDir + "/.k9s-komodor-rca"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	clusterMappingFile := configDir + "/clusters.yaml"
	data, err := yaml.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal cluster mapping: %w", err)
	}

	if err := os.WriteFile(clusterMappingFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cluster mapping file: %w", err)
	}

	return nil
}

func getLocalClusterUID() (string, error) {
	cmd := exec.Command("kubectl", "get", "namespace", "default", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get cluster UID: %w", err)
	}

	var namespaceData map[string]interface{}
	if err := json.Unmarshal(output, &namespaceData); err != nil {
		return "", fmt.Errorf("failed to parse kubectl output: %w", err)
	}

	metadata, ok := namespaceData["metadata"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid namespace metadata structure")
	}

	uid, ok := metadata["uid"].(string)
	if !ok {
		return "", fmt.Errorf("cluster UID not found in namespace metadata")
	}

	return uid, nil
}

func resolveKomodorCluster(apiKey, baseURL, localClusterName string) (string, error) {
	mapping, err := loadClusterMapping()
	if err != nil {
		mapping = &ClusterMapping{Mapping: make(map[string]string)}
	}

	if komodorCluster, exists := mapping.Mapping[localClusterName]; exists {
		logMessage("Using mapped Komodor cluster '%s' for local cluster '%s'", komodorCluster, localClusterName)
		return komodorCluster, nil
	}

	logMessage("No mapping found for cluster '%s', attempting to fetch Komodor clusters", localClusterName)
	komodorClusters, err := fetchKomodorClusters(apiKey, baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Komodor clusters: %w", err)
	}

	matchingCluster := findMatchingClusterByName(localClusterName, komodorClusters)
	if matchingCluster == nil {
		logMessage("⚠️  No name match found, trying to match by cluster UID")
		localClusterUID, err := getLocalClusterUID()
		if err == nil {
			matchingCluster = findMatchingClusterByUID(localClusterUID, komodorClusters)
		} else {
			logMessage("⚠️  Could not get local cluster UID: %v", err)
		}
	}

	if matchingCluster != nil {
		logMessage("✅ Found matching Komodor cluster: '%s'", matchingCluster.Name)
		mapping.Mapping[localClusterName] = matchingCluster.Name
		if err := saveClusterMapping(mapping); err != nil {
			logMessage("⚠️  Could not save cluster mapping: %v", err)
		} else {
			logMessage("💾 Saved mapping: '%s' -> '%s'", localClusterName, matchingCluster.Name)
		}
		return matchingCluster.Name, nil
	}

	logMessage("ERROR: No matching Komodor cluster found for '%s'", localClusterName)
	return "", fmt.Errorf("no matching Komodor cluster found for '%s'. Available clusters: %s\n\n💡 To fix this, add a manual mapping to ~/.k9s-komodor-rca/clusters.yaml:\nmapping:\n  \"%s\": \"your-komodor-cluster-name\"",
		localClusterName, getClusterNames(komodorClusters), localClusterName)
}

func findMatchingClusterByName(k9sClusterName string, komodorClusters []KomodorCluster) *KomodorCluster {
	for _, cluster := range komodorClusters {
		if cluster.Name == k9sClusterName {
			return &cluster
		}
	}
	return nil
}

func findMatchingClusterByUID(localClusterUID string, komodorClusters []KomodorCluster) *KomodorCluster {
	for _, cluster := range komodorClusters {
		if cluster.ClusterID == localClusterUID {
			return &cluster
		}
	}
	return nil
}

func getClusterNames(komodorClusters []KomodorCluster) string {
	names := make([]string, len(komodorClusters))
	for i, cluster := range komodorClusters {
		names[i] = cluster.Name
	}
	return strings.Join(names, ", ")
}
