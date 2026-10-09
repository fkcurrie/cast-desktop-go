#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

PKG_NAME="cast-desktop-go"
PKG_VERSION="1.2.0"
PKG_ARCH="amd64"
DEB_NAME="${PKG_NAME}_${PKG_VERSION}_${PKG_ARCH}.deb"
BUILD_ROOT="${ROOT_DIR}/build/deb-pkg"
OUTPUT_DIR="${ROOT_DIR}/dist"

echo "=== Building ${PKG_NAME} v${PKG_VERSION} Debian Package ==="

# 1. Compile release binary
echo "--> Compiling Go binary..."
mkdir -p "${ROOT_DIR}/bin" "${OUTPUT_DIR}"
cd "${ROOT_DIR}"
CGO_ENABLED=0 go build -ldflags="-s -w -X 'cast-desktop-go/cmd.Version=${PKG_VERSION}'" -o "${ROOT_DIR}/bin/cast-desktop-go" main.go

# 2. Prepare staging tree
echo "--> Setting up staging directory structure in ${BUILD_ROOT}..."
rm -rf "${BUILD_ROOT}"
mkdir -p "${BUILD_ROOT}/DEBIAN"
mkdir -p "${BUILD_ROOT}/usr/bin"
mkdir -p "${BUILD_ROOT}/usr/share/cast-desktop-go"
mkdir -p "${BUILD_ROOT}/usr/share/applications"
mkdir -p "${BUILD_ROOT}/usr/share/metainfo"
mkdir -p "${BUILD_ROOT}/usr/share/man/man1"
mkdir -p "${BUILD_ROOT}/usr/share/doc/${PKG_NAME}"
mkdir -p "${BUILD_ROOT}/usr/share/icons/hicolor/scalable/apps"

# 3. Write DEBIAN control file
cat <<EOF > "${BUILD_ROOT}/DEBIAN/control"
Package: ${PKG_NAME}
Version: ${PKG_VERSION}
Section: video
Priority: optional
Architecture: ${PKG_ARCH}
Maintainer: Frank Currie <fkcurrie@users.noreply.github.com>
Depends: ffmpeg, python3, python3-gi, python3-cairo, gir1.2-gtk-3.0, libnotify-bin, kdialog | zenity
Recommends: va-driver-all, mesa-va-drivers
Homepage: https://github.com/fkcurrie/cast-desktop-go
Description: Chromecast Desktop Streamer for Linux
 High-performance screen and audio casting utility for Google Chromecast
 devices with AMD VA-API hardware acceleration, web remote, tray icon,
 latency presets, monitor/window selector, privacy border, and volume controls.
EOF
chmod 644 "${BUILD_ROOT}/DEBIAN/control"

# 4. Write postinst script
cat <<'EOF' > "${BUILD_ROOT}/DEBIAN/postinst"
#!/bin/sh
set -e

if which gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi

if which update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q /usr/share/applications || true
fi

# Refresh KDE Plasma system configuration cache if available
if which kbuildsycoca6 >/dev/null 2>&1; then
    XDG_MENU_PREFIX="${XDG_MENU_PREFIX:-plasma-}" kbuildsycoca6 --noincremental >/dev/null 2>&1 || true
elif which kbuildsycoca5 >/dev/null 2>&1; then
    kbuildsycoca5 --noincremental >/dev/null 2>&1 || true
fi

exit 0
EOF
chmod 755 "${BUILD_ROOT}/DEBIAN/postinst"

# 5. Write postrm script
cat <<'EOF' > "${BUILD_ROOT}/DEBIAN/postrm"
#!/bin/sh
set -e

if which gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi

if which update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q /usr/share/applications || true
fi

if which kbuildsycoca6 >/dev/null 2>&1; then
    XDG_MENU_PREFIX="${XDG_MENU_PREFIX:-plasma-}" kbuildsycoca6 --noincremental >/dev/null 2>&1 || true
elif which kbuildsycoca5 >/dev/null 2>&1; then
    kbuildsycoca5 --noincremental >/dev/null 2>&1 || true
fi

exit 0
EOF
chmod 755 "${BUILD_ROOT}/DEBIAN/postrm"

# 6. Install binaries
echo "--> Installing binaries..."
cp "${ROOT_DIR}/bin/cast-desktop-go" "${BUILD_ROOT}/usr/bin/cast-desktop-go"
chmod 755 "${BUILD_ROOT}/usr/bin/cast-desktop-go"
ln -sf "cast-desktop-go" "${BUILD_ROOT}/usr/bin/cast-go"

# 7. Install desktop launcher
echo "--> Installing desktop file..."
cp "${ROOT_DIR}/assets/cast-desktop.desktop" "${BUILD_ROOT}/usr/share/applications/cast-desktop.desktop"
chmod 644 "${BUILD_ROOT}/usr/share/applications/cast-desktop.desktop"

# 8. Install AppStream metainfo
echo "--> Installing AppStream metadata..."
cp "${ROOT_DIR}/assets/io.github.fkcurrie.cast-desktop-go.metainfo.xml" "${BUILD_ROOT}/usr/share/metainfo/io.github.fkcurrie.cast-desktop-go.metainfo.xml"
chmod 644 "${BUILD_ROOT}/usr/share/metainfo/io.github.fkcurrie.cast-desktop-go.metainfo.xml"

# 9. Install documentation, changelog, and copyright
echo "--> Installing documentation, copyright and changelog..."
cp "${ROOT_DIR}/debian/copyright" "${BUILD_ROOT}/usr/share/doc/${PKG_NAME}/copyright"
chmod 644 "${BUILD_ROOT}/usr/share/doc/${PKG_NAME}/copyright"
gzip -9nc "${ROOT_DIR}/debian/changelog" > "${BUILD_ROOT}/usr/share/doc/${PKG_NAME}/changelog.gz"
chmod 644 "${BUILD_ROOT}/usr/share/doc/${PKG_NAME}/changelog.gz"

# 10. Install manpage
echo "--> Installing manual page..."
gzip -9nc "${ROOT_DIR}/assets/cast-desktop-go.1" > "${BUILD_ROOT}/usr/share/man/man1/cast-desktop-go.1.gz"
ln -sf "cast-desktop-go.1.gz" "${BUILD_ROOT}/usr/share/man/man1/cast-go.1.gz"
chmod 644 "${BUILD_ROOT}/usr/share/man/man1/cast-desktop-go.1.gz"

# 11. Install icons
echo "--> Installing icons..."
cp "${ROOT_DIR}/assets/icons/scalable/cast-desktop.svg" "${BUILD_ROOT}/usr/share/icons/hicolor/scalable/apps/cast-desktop.svg"
ln -sf "cast-desktop.svg" "${BUILD_ROOT}/usr/share/icons/hicolor/scalable/apps/cast-desktop-go.svg"
chmod 644 "${BUILD_ROOT}/usr/share/icons/hicolor/scalable/apps/cast-desktop.svg"

for size in 16x16 24x24 32x32 48x48 64x64 128x128 256x256 512x512; do
    if [ -f "${ROOT_DIR}/assets/icons/${size}/cast-desktop.png" ]; then
        mkdir -p "${BUILD_ROOT}/usr/share/icons/hicolor/${size}/apps"
        cp "${ROOT_DIR}/assets/icons/${size}/cast-desktop.png" "${BUILD_ROOT}/usr/share/icons/hicolor/${size}/apps/cast-desktop.png"
        ln -sf "cast-desktop.png" "${BUILD_ROOT}/usr/share/icons/hicolor/${size}/apps/cast-desktop-go.png"
        chmod 644 "${BUILD_ROOT}/usr/share/icons/hicolor/${size}/apps/cast-desktop.png"
    fi
done

# 12. Install helper scripts (privacy border overlay)
echo "--> Installing overlay scripts..."
cp "${ROOT_DIR}/scripts/border_overlay.py" "${BUILD_ROOT}/usr/share/cast-desktop-go/border_overlay.py"
chmod 755 "${BUILD_ROOT}/usr/share/cast-desktop-go/border_overlay.py"

# 13. Ensure standard directory and file permissions
find "${BUILD_ROOT}" -type d -exec chmod 0755 {} +

# 14. Build Debian package
echo "--> Packaging with dpkg-deb..."
dpkg-deb --root-owner-group --build "${BUILD_ROOT}" "${OUTPUT_DIR}/${DEB_NAME}"

echo "=== Package Built Successfully! ==="
echo "Artifact: ${OUTPUT_DIR}/${DEB_NAME}"
ls -lh "${OUTPUT_DIR}/${DEB_NAME}"
