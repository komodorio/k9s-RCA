#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="k9s-rca"
VERSION=${VERSION:-$(git describe --tags --always --dirty)}
BUILD_DIR="dist"
RELEASE_DIR="release"

echo -e "${GREEN}Building k9s-rca v${VERSION}${NC}"

# Clean previous builds
rm -rf ${BUILD_DIR} ${RELEASE_DIR}
mkdir -p ${BUILD_DIR} ${RELEASE_DIR}

# Build for multiple platforms
echo -e "${YELLOW}Building for multiple platforms...${NC}"

# Linux AMD64
echo "Building for Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 .

# Linux ARM64
echo "Building for Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=${VERSION}" -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 .

# macOS AMD64
echo "Building for macOS AMD64..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 .

# macOS ARM64 (Apple Silicon)
echo "Building for macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=${VERSION}" -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 .

# Windows AMD64
echo "Building for Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe .

# Create checksums
echo -e "${YELLOW}Creating checksums...${NC}"
cd ${BUILD_DIR}
for file in *; do
    if [[ -f "$file" ]]; then
        sha256sum "$file" > "${file}.sha256"
    fi
done
cd ..

# Create release archive
echo -e "${YELLOW}Creating release archive...${NC}"
tar -czf ${RELEASE_DIR}/k9s-rca-${VERSION}-linux-amd64.tar.gz -C ${BUILD_DIR} ${BINARY_NAME}-linux-amd64 ${BINARY_NAME}-linux-amd64.sha256
tar -czf ${RELEASE_DIR}/k9s-rca-${VERSION}-linux-arm64.tar.gz -C ${BUILD_DIR} ${BINARY_NAME}-linux-arm64 ${BINARY_NAME}-linux-arm64.sha256
tar -czf ${RELEASE_DIR}/k9s-rca-${VERSION}-darwin-amd64.tar.gz -C ${BUILD_DIR} ${BINARY_NAME}-darwin-amd64 ${BINARY_NAME}-darwin-amd64.sha256
tar -czf ${RELEASE_DIR}/k9s-rca-${VERSION}-darwin-arm64.tar.gz -C ${BUILD_DIR} ${BINARY_NAME}-darwin-arm64 ${BINARY_NAME}-darwin-arm64.sha256
zip -j ${RELEASE_DIR}/k9s-rca-${VERSION}-windows-amd64.zip ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe.sha256

# Create install script
echo -e "${YELLOW}Creating install script...${NC}"
cat > ${RELEASE_DIR}/install.sh << 'EOF'
#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

BINARY_NAME="k9s-rca"
INSTALL_DIR="${HOME}/.local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $ARCH${NC}" && exit 1 ;;
esac

# Determine binary name
if [[ "$OS" == "darwin" ]]; then
    OS="darwin"
elif [[ "$OS" == "linux" ]]; then
    OS="linux"
else
    echo -e "${RED}Unsupported OS: $OS${NC}" && exit 1
fi

BINARY_FILE="${BINARY_NAME}-${OS}-${ARCH}"
if [[ "$OS" == "windows" ]]; then
    BINARY_FILE="${BINARY_FILE}.exe"
fi

echo -e "${GREEN}Installing k9s-rca for ${OS}-${ARCH}${NC}"

# Create install directory
mkdir -p "${INSTALL_DIR}"

# Download and install binary
echo -e "${YELLOW}Downloading binary...${NC}"
curl -L -o "${INSTALL_DIR}/${BINARY_NAME}" "https://github.com/komodorio/k9s-RCA/releases/latest/download/${BINARY_FILE}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

echo -e "${GREEN}Installation complete!${NC}"
echo -e "${YELLOW}Binary installed to: ${INSTALL_DIR}/${BINARY_NAME}${NC}"
echo -e "${YELLOW}Make sure ${INSTALL_DIR} is in your PATH${NC}"
EOF

chmod +x ${RELEASE_DIR}/install.sh

# Create plugin installation script
echo -e "${YELLOW}Creating plugin installation script...${NC}"
cat > ${RELEASE_DIR}/install-plugin.sh << 'EOF'
#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

K9S_CONFIG_DIR="${HOME}/.config/k9s"
PLUGIN_FILE="k9s_rca_plugin.yaml"

echo -e "${GREEN}Installing k9s-rca plugin configuration...${NC}"

# Create k9s config directory
mkdir -p "${K9S_CONFIG_DIR}"

# Copy plugin configuration
if [[ -f "${PLUGIN_FILE}" ]]; then
    cp "${PLUGIN_FILE}" "${K9S_CONFIG_DIR}/plugins.yaml"
    echo -e "${GREEN}Plugin configuration installed to ${K9S_CONFIG_DIR}/plugins.yaml${NC}"
else
    echo -e "${RED}Plugin configuration file not found: ${PLUGIN_FILE}${NC}"
    exit 1
fi

echo -e "${GREEN}Plugin installation complete!${NC}"
echo -e "${YELLOW}Restart k9s to load the plugin${NC}"
EOF

chmod +x ${RELEASE_DIR}/install-plugin.sh

# Summary
echo -e "${GREEN}Build complete!${NC}"
echo -e "${YELLOW}Binaries created in: ${BUILD_DIR}/${NC}"
echo -e "${YELLOW}Release archives in: ${RELEASE_DIR}/${NC}"
echo -e "${YELLOW}Version: ${VERSION}${NC}"

# List all created files
echo -e "\n${GREEN}Created files:${NC}"
ls -la ${BUILD_DIR}/
echo -e "\n${GREEN}Release archives:${NC}"
ls -la ${RELEASE_DIR}/

echo -e "\n${YELLOW}To create a GitHub release:${NC}"
echo "1. Create a new release on GitHub"
echo "2. Upload the files from ${RELEASE_DIR}/"
echo "3. Tag the release with v${VERSION}" 