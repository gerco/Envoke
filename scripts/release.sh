#!/usr/bin/env bash
# Creates release archives, checksums, a GitHub release, and updates the
# Homebrew formula. Expects pre-built binaries in dist/<os>_<arch>/.
#
# Required env vars:
#   GITHUB_REF_NAME          — the tag, e.g. v1.2.3
#   GH_TOKEN                 — GitHub token for gh CLI (release creation)
#   HOMEBREW_TAP_GITHUB_TOKEN — GitHub token with write access to gerco/homebrew-tap
set -euo pipefail

TAG="${GITHUB_REF_NAME}"
VERSION="${TAG#v}"
REPO="gerco/envoke"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

# ---------------------------------------------------------------------------
# Archives
# ---------------------------------------------------------------------------
mkdir -p release

for OS in darwin linux; do
  for ARCH in amd64 arm64; do
    tar -czf "release/envoke_${VERSION}_${OS}_${ARCH}.tar.gz" \
      -C "dist/${OS}_${ARCH}" envoke
  done
done

for ARCH in amd64 arm64; do
  (cd "dist/windows_${ARCH}" && zip "../../release/envoke_${VERSION}_windows_${ARCH}.zip" envoke.exe)
done

(cd release && sha256sum ./* > checksums.txt)

# ---------------------------------------------------------------------------
# GitHub release
# ---------------------------------------------------------------------------
gh release create "${TAG}" \
  --title "${TAG}" \
  --generate-notes \
  release/*

# ---------------------------------------------------------------------------
# Homebrew formula
# ---------------------------------------------------------------------------
sha_darwin_amd64=$(sha256sum "release/envoke_${VERSION}_darwin_amd64.tar.gz" | awk '{print $1}')
sha_darwin_arm64=$(sha256sum "release/envoke_${VERSION}_darwin_arm64.tar.gz" | awk '{print $1}')
sha_linux_amd64=$(sha256sum  "release/envoke_${VERSION}_linux_amd64.tar.gz"  | awk '{print $1}')
sha_linux_arm64=$(sha256sum  "release/envoke_${VERSION}_linux_arm64.tar.gz"  | awk '{print $1}')

cat > /tmp/envoke.rb << FORMULA
class Envoke < Formula
  desc "Environment secrets manager with pluggable backends"
  homepage "https://git.dries.info/gerco/Envoke"
  license "MIT"
  version "${VERSION}"

  on_macos do
    on_intel do
      url "${BASE_URL}/envoke_${VERSION}_darwin_amd64.tar.gz"
      sha256 "${sha_darwin_amd64}"
    end
    on_arm do
      url "${BASE_URL}/envoke_${VERSION}_darwin_arm64.tar.gz"
      sha256 "${sha_darwin_arm64}"
    end
  end

  on_linux do
    on_intel do
      url "${BASE_URL}/envoke_${VERSION}_linux_amd64.tar.gz"
      sha256 "${sha_linux_amd64}"
    end
    on_arm do
      url "${BASE_URL}/envoke_${VERSION}_linux_arm64.tar.gz"
      sha256 "${sha_linux_arm64}"
    end
  end

  def install
    bin.install "envoke"
    bin.install_symlink "envoke" => "ee"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/envoke version")
    assert_match version.to_s, shell_output("#{bin}/ee version")
  end
end
FORMULA

git clone \
  "https://x-access-token:${HOMEBREW_TAP_GITHUB_TOKEN}@github.com/gerco/homebrew-tap.git" \
  /tmp/homebrew-tap
cp /tmp/envoke.rb /tmp/homebrew-tap/Formula/envoke.rb
cd /tmp/homebrew-tap
git config user.email "github-actions@github.com"
git config user.name "GitHub Actions"
git add Formula/envoke.rb
git commit -m "Update envoke to ${TAG}"
git push
