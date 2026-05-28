class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.5.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.0/oops_darwin_arm64.tar.gz"
      sha256 "f49101a26c35f1c161711609ad94a0ae4790f76860107ff7677320e180208d24"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.0/oops_darwin_amd64.tar.gz"
      sha256 "2fa51dec4abe0d493859b2461a22e35e9b66b71fd55229e6fcf985f9e503e118"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.0/oops_linux_arm64.tar.gz"
      sha256 "3feaaf0ab353f46a77e762758c89e85291b5f7a7711fa607ee3a0ae7ddc94e67"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.0/oops_linux_amd64.tar.gz"
      sha256 "04985df74949175dca34c98afd1e874fe7cd25e6b63ea11ca38e10aedcaeea82"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
