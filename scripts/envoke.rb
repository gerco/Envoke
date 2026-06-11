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
