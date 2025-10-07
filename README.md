# K9s Komodor RCA Plugin

A K9s plugin that integrates Komodor's Root Cause Analysis directly into your Kubernetes workflow. Trigger RCA analysis for any Kubernetes resource with `Shift-K` while browsing your cluster.

![K9s RCA Plugin Demo](k9s-rca.gif)

## Prerequisites

- **Komodor Account**: Sign up at [komodor.com](https://komodor.com)
- **API Key**: Generate from Komodor dashboard → Settings → API Keys
- **K9s**: Install from [k9scli.io](https://k9scli.io/topics/install/)

## Installation

### Important: K9s Plugin Configuration

**For the plugin to work, K9s must find the plugin configuration file.** K9s looks for plugins in:
- `$XDG_CONFIG_HOME/k9s/plugins.yaml` (if `XDG_CONFIG_HOME` is set)
- `~/.config/k9s/plugins.yaml` (default)

If you have issues, set `XDG_CONFIG_HOME`:
```bash
export XDG_CONFIG_HOME="$HOME/.config"
```
Add this to your `~/.bashrc` or `~/.zshrc` to make it permanent.

### Homebrew (macOS/Linux)

```bash
# Add the tap and install
brew tap komodorio/k9s-rca https://github.com/komodorio/k9s-rca
brew install k9s-rca

# Copy plugin configuration to k9s (REQUIRED)
mkdir -p ~/.config/k9s
cp $(brew --prefix)/share/k9s-rca/k9s_rca_plugin.yaml ~/.config/k9s/plugins.yaml

# Restart k9s if it's running
pkill k9s
```

### Prebuilt Binaries

Download from [GitHub Releases](https://github.com/komodorio/k9s-rca/releases/latest):

```bash
# Download and extract (replace VERSION, OS, and ARCH as needed)
VERSION=1.0.0
OS=darwin  # or linux, windows
ARCH=arm64 # or amd64
curl -L -o k9s-rca.tar.gz "https://github.com/komodorio/k9s-rca/releases/download/v${VERSION}/k9s-rca-${VERSION}-${OS}-${ARCH}.tar.gz"
tar -xzf k9s-rca.tar.gz

# Install binary
sudo mv k9s-rca /usr/local/bin/

# Copy plugin configuration (REQUIRED)
mkdir -p ~/.config/k9s
mv k9s_rca_plugin.yaml ~/.config/k9s/plugins.yaml

# Restart k9s if it's running
pkill k9s
```

### Build from Source

```bash
git clone https://github.com/komodorio/k9s-rca.git
cd k9s-rca
make install-plugin  # Builds binary and copies plugin config
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

The plugin configuration MUST be in the correct location for K9s to find it.

```bash
# Check if XDG_CONFIG_HOME is set
echo $XDG_CONFIG_HOME

# If not set, set it and add to your shell profile
export XDG_CONFIG_HOME="$HOME/.config"
echo 'export XDG_CONFIG_HOME="$HOME/.config"' >> ~/.zshrc  # or ~/.bashrc

# Verify plugin config exists
ls -la ~/.config/k9s/plugins.yaml

# Restart k9s completely
pkill k9s
k9s
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
