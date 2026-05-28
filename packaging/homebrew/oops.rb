class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.5.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.2/oops_darwin_arm64.tar.gz"
      sha256 "525c8e940e9b67c7f43bfe57b1d52e4f5bd3dba9956fe0f47bd4624452e9913c"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.2/oops_darwin_amd64.tar.gz"
      sha256 "d562c3794e05c86fa406ba38dc35f9af5b13358e97bac6c3b4363f8a21a327b8"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.2/oops_linux_arm64.tar.gz"
      sha256 "86d2abb6550c140499fb5502a61a4e599cc073117bca425c3c60ac65f91d4521"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.2/oops_linux_amd64.tar.gz"
      sha256 "4586e199d41962ab9be8ac7854b50c1c5dde31d18d2860a512c72c731ec9864e"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
