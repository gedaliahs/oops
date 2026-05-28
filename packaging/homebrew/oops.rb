class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.5.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.1/oops_darwin_arm64.tar.gz"
      sha256 "9f95b16b92af9670a8e5fc308abe42cd1ba8fa833c329096c2d4ced336e49ffb"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.1/oops_darwin_amd64.tar.gz"
      sha256 "f9d2b43e50391e3b763bd66e7033538bd8c8bd65fb29c5a7f74f2d9a09e758e8"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.1/oops_linux_arm64.tar.gz"
      sha256 "a2ccd37b8f467204826ff1762210379be6eebddf4ee50ebcc423db7c7f61c8c7"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.5.1/oops_linux_amd64.tar.gz"
      sha256 "0e5d83439b80367cca3add1617ad01b0c9062a7ac048ff118c7799d4b2c053a1"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
