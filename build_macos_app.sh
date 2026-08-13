#!/usr/bin/env bash
set -euo pipefail

APP_NAME="YTTranscriber"
BUILD_DIR="build"
APP_DIR="${BUILD_DIR}/${APP_NAME}.app"
BIN_DIR="${APP_DIR}/Contents/MacOS"
RES_DIR="${APP_DIR}/Contents/Resources"

mkdir -p "${BIN_DIR}" "${RES_DIR}"

go build -o "${BIN_DIR}/${APP_NAME}" .
cp "assets/logo.svg" "${RES_DIR}/logo.svg"

cat > "${APP_DIR}/Contents/Info.plist" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>YTTranscriber</string>
  <key>CFBundleDisplayName</key>
  <string>YTTranscriber</string>
  <key>CFBundleIdentifier</key>
  <string>local.yt.transcriber</string>
  <key>CFBundleVersion</key>
  <string>1.0.0</string>
  <key>CFBundleShortVersionString</key>
  <string>1.0.0</string>
  <key>CFBundleExecutable</key>
  <string>YTTranscriber</string>
  <key>CFBundleIconFile</key>
  <string>logo.svg</string>
  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>
</dict>
</plist>
EOF

echo "Built: ${APP_DIR}"
