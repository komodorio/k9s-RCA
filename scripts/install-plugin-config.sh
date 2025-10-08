#!/bin/bash
set -e

echo "🔧 K9s RCA Plugin Configuration Installer"
echo ""

# Determine k9s config directory
if [ -n "$XDG_CONFIG_HOME" ]; then
    K9S_CONFIG_DIR="$XDG_CONFIG_HOME/k9s"
    echo "✓ Using XDG_CONFIG_HOME: $XDG_CONFIG_HOME"
else
    K9S_CONFIG_DIR="$HOME/.config/k9s"
    echo "⚠️  XDG_CONFIG_HOME not set, using default: $K9S_CONFIG_DIR"
    echo "   Consider adding to your shell profile:"
    echo "   export XDG_CONFIG_HOME=\"\$HOME/.config\""
fi

echo ""

# Find the plugin yaml file
PLUGIN_YAML=""
if [ -f "k9s_rca_plugin.yaml" ]; then
    PLUGIN_YAML="k9s_rca_plugin.yaml"
elif [ -f "$(brew --prefix 2>/dev/null)/share/k9s-rca/k9s_rca_plugin.yaml" ]; then
    PLUGIN_YAML="$(brew --prefix)/share/k9s-rca/k9s_rca_plugin.yaml"
elif [ -f "../k9s_rca_plugin.yaml" ]; then
    PLUGIN_YAML="../k9s_rca_plugin.yaml"
else
    echo "❌ Error: Could not find k9s_rca_plugin.yaml"
    echo "   Please run this script from the k9s-rca directory or after installing via Homebrew"
    exit 1
fi

echo "📄 Found plugin config: $PLUGIN_YAML"
echo ""

# Create config directory
echo "📁 Creating config directory: $K9S_CONFIG_DIR"
mkdir -p "$K9S_CONFIG_DIR"

# Backup existing plugins.yaml if it exists
if [ -f "$K9S_CONFIG_DIR/plugins.yaml" ]; then
    BACKUP_FILE="$K9S_CONFIG_DIR/plugins.yaml.backup.$(date +%Y%m%d-%H%M%S)"
    echo "⚠️  Existing plugins.yaml found"
    echo "   Creating backup: $BACKUP_FILE"
    cp "$K9S_CONFIG_DIR/plugins.yaml" "$BACKUP_FILE"
    echo ""
fi

# Copy plugin configuration
echo "📋 Installing plugin configuration to: $K9S_CONFIG_DIR/plugins.yaml"
cp "$PLUGIN_YAML" "$K9S_CONFIG_DIR/plugins.yaml"

echo ""
echo "✅ Plugin configuration installed successfully!"
echo ""
echo "📋 Next steps:"
echo ""
echo "1. Set your Komodor API key (REQUIRED):"
echo "   export KOMODOR_API_KEY=\"your-api-key\""
echo "   Add to ~/.zshrc or ~/.bashrc to make it permanent"
echo ""
echo "2. Verify k9s-rca binary is installed:"
echo "   which k9s-rca"
echo ""
echo "3. Restart k9s (if running):"
echo "   pkill k9s && k9s"
echo ""
echo "4. In k9s, press Shift-K on any resource to trigger RCA"
echo ""

