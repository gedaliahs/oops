class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.4.9"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_darwin_arm64.tar.gz"
      sha256 "f8e0205fbfc587d2abf2bff1a5dc7f8243ae99422ddd7b41470c9b565045f7b2"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_darwin_amd64.tar.gz"
      sha256 "919f7203dd685bed13052d3bc66e75f5abcdf5c7fbe4b4f2934c68f6af1fc731"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_linux_arm64.tar.gz"
      sha256 "0db286f07f5717d2cc51fc4341fc575375fb1506a4919cc424162d4a015dbfe7"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.4.9/oops_linux_amd64.tar.gz"
      sha256 "c01b6b31829c87aa93625a790faf4cda71eb61dd02da451fa6ed16fb7e42ab45"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
