class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.4.9"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_darwin_arm64.tar.gz"
      sha256 "f918fd32a2f1ae8fc93e64bb510b82e84f5f7a8a43d31e2be9559e2faacac4b1"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_darwin_amd64.tar.gz"
      sha256 "79435972f3bb599e78907df0d95c792bbcf9cb15c65ecd28603efdd9f5e944c7"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_linux_arm64.tar.gz"
      sha256 "4c007c84325bb806a20ff294d392e7859c4c563acadb25128a69fffde7afdeb1"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_linux_amd64.tar.gz"
      sha256 "549f8243944408c513603d1d7b10a3d953861a537a3053aecb87acc2b36e12d6"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
