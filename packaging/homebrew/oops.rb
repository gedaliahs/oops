class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.5.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.3/oops_darwin_arm64.tar.gz"
      sha256 "6b15faa14940f617c20402947dad131a85f51930cff5267a6fc61d11bcd10779"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.3/oops_darwin_amd64.tar.gz"
      sha256 "e5bb90010debbf7d52be8d337c1b0b4d8be1dbf6450260179805dc51010667c0"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.3/oops_linux_arm64.tar.gz"
      sha256 "28858d283a86fa665d7b3aa711de8fefd313f95b5f88702034bf2b491e3eff87"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.3/oops_linux_amd64.tar.gz"
      sha256 "a0a2e81bf5f867d16f12c0eea2711387252192e32bef95b6fd3500290d1e8db4"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
