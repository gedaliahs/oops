class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.5.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.0/oops_darwin_arm64.tar.gz"
      sha256 "bcc6cf2a2827c381b5cea23fca15816da5eb5ed457a72ded0d1b5a172d0faeb1"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.0/oops_darwin_amd64.tar.gz"
      sha256 "0954390f8438a43dfe5a5bf09429740bac6e744bad80bfcc2dd0ad5bbab2d35b"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.0/oops_linux_arm64.tar.gz"
      sha256 "b33362bcbf6142ad8ab76162b00609efbfacd8d6e017216dc639e0792fe0c718"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.0/oops_linux_amd64.tar.gz"
      sha256 "b0e2a246a63966ef70e1c1bb3c44bddde5552fa28ec2a16222a2484fa21a1e7b"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
