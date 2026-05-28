class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.4.9"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_darwin_arm64.tar.gz"
      sha256 "d651ee7f2ecbff7b791d1b8727db836c696a0c2a01bc56fb5a30719749996969"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_darwin_amd64.tar.gz"
      sha256 "fb59fca174932efc1d2d35fa2ce42f5517e352c1262fa71b16af9142ed508636"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_linux_arm64.tar.gz"
      sha256 "be37cf7a384aa04e51bde25fabc825916192e816c6338733c201779fe3a367e8"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_linux_amd64.tar.gz"
      sha256 "04105a0c1f06f1b9202b18877806e789a9d932dda9b6d14fb584d14f4333e76b"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
