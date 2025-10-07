# K9s Komodor RCA Plugin

A K9s plugin that integrates Komodor's Root Cause Analysis directly into your Kubernetes workflow. Trigger RCA analysis for any Kubernetes resource with `Shift-K` while browsing your cluster.

![K9s RCA Plugin Demo](demo-k9s-rca.gif)

## Prerequisites

- **Komodor Account**: Sign up at [komodor.com](https://komodor.com)
- **API Key**: Generate from Komodor dashboard → Settings → API Keys
- **K9s**: Install from [k9scli.io](https://k9scli.io/topics/install/)

## Installation

### Homebrew (macOS/Linux)

```bash
brew install komodorio/tap/k9s-rca
mkdir -p ~/.config/k9s
cp $(brew --prefix)/share/k9s-rca/k9s_rca_plugin.yaml ~/.config/k9s/plugins.yaml
```

### Prebuilt Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/komodorio/k9s-rca/releases):

```bash
VERSION=v1.0.0
OS=darwin  # or linux, windows
ARCH=arm64  # or amd64

curl -L -o k9s-rca.tar.gz \
  "https://github.com/komodorio/k9s-rca/releases/download/${VERSION}/k9s-rca-${VERSION#v}-${OS}-${ARCH}.tar.gz"
tar -xzf k9s-rca.tar.gz
sudo mv k9s-rca /usr/local/bin/
mkdir -p ~/.config/k9s
mv k9s_rca_plugin.yaml ~/.config/k9s/plugins.yaml
```

### Build from Source

```bash
git clone https://github.com/komodorio/k9s-rca.git
cd k9s-rca
make install-plugin
```

## Configuration

Set your Komodor API key:

```bash
export KOMODOR_API_KEY="your-api-key-here"
```

Add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) to persist across sessions.

### Cluster Mapping (Optional)

If your local cluster name differs from Komodor's cluster name, create `~/.k9s-komodor-rca/clusters.yaml`:

```yaml
mapping:
  "minikube": "production-cluster-1"
  "docker-desktop": "staging-cluster"
  "gke_project_zone_cluster": "gke-production"
```

## Usage

1. Open K9s: `k9s`
2. Navigate to any resource (`:po`, `:deploy`, `:svc`, etc.)
3. Select a resource with arrow keys
4. Press `Shift-K` to trigger RCA

### Supported Resources

Pods, Deployments, Services, StatefulSets, DaemonSets, Ingress, ConfigMaps, Secrets, PersistentVolumeClaims, Jobs, CronJobs, ReplicaSets, HorizontalPodAutoscalers, PodDisruptionBudgets, NetworkPolicies

## Command Line Options

```bash
k9s-rca --help
```

Available flags:
- `--kind`: Resource kind (Pod, Deployment, etc.)
- `--namespace`: Namespace
- `--name`: Resource name
- `--api-key`: Komodor API key (overrides env var)
- `--cluster`: Cluster name
- `--base-url`: API base URL (default: https://api.komodor.com)
- `--poll`: Monitor RCA completion
- `--background`: Run without TUI
- `--debug`: Enable debug logging to `~/.k9s-komodor-rca/k9s_komodor_logs.txt`

## Troubleshooting

**Plugin not loading:**
```bash
ls -la ~/.config/k9s/plugins.yaml
pkill k9s && k9s
```

**Binary not found:**
```bash
which k9s-rca
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

**API errors:**
Check logs with `--debug` flag or view `~/.k9s-komodor-rca/k9s_komodor_logs.txt`

## Development

```bash
git clone https://github.com/komodorio/k9s-rca.git
cd k9s-rca
make build          # Build binary
make test           # Run tests
make install        # Install to ~/.local/bin
make clean          # Clean build artifacts
```

### Creating a Release

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

GitHub Actions will automatically build and publish the release with GoReleaser.

## License

MIT License - See [LICENSE](LICENSE) file for details.

---

**Documentation**: [Komodor Help Center](https://help.komodor.com/) | **API Docs**: [api.komodor.com](https://api.komodor.com/api/docs/)
