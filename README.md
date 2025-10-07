# K9s Komodor RCA Plugin

A K9s plugin that integrates Komodor's Root Cause Analysis directly into your Kubernetes workflow. Trigger RCA analysis for any Kubernetes resource with `Shift-K` while browsing your cluster.

![K9s RCA Plugin Demo](k9s-rca.gif)

## Prerequisites

- **Komodor Account**: Sign up at [komodor.com](https://komodor.com)
- **API Key**: Generate from Komodor dashboard → Settings → API Keys
- **K9s**: Install from [k9scli.io](https://k9scli.io/topics/install/)

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap komodorio/k9s-rca https://github.com/komodorio/k9s-rca
brew install k9s-rca
mkdir -p ~/.config/k9s
cp $(brew --prefix)/share/k9s-rca/k9s_rca_plugin.yaml ~/.config/k9s/plugins.yaml
```

### Build from Source

```bash
git clone https://github.com/komodorio/k9s-rca.git
cd k9s-rca
make install-plugin
```

## Configuration

Set your Komodor API key using either method:

**Environment Variable:**

```bash
export KOMODOR_API_KEY="your-api-key-here"
```

Add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) to persist across sessions.

**`.env` File:**
Create a `.env` file in your project directory or `~/.k9s-komodor-rca/.env`:
```bash
KOMODOR_API_KEY=your-api-key-here
```

### Cluster Mapping (Optional)

The plugin automatically detects and matches your cluster name with Komodor. In rare cases where auto-detection fails, you can manually configure cluster name mapping by creating `~/.k9s-komodor-rca/clusters.yaml`:

```yaml
mapping:
  "local-cluster-name": "komodor-cluster-name"
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

## License

MIT License - See [LICENSE](LICENSE) file for details.

---

**Documentation**: [Komodor Help Center](https://help.komodor.com/)
