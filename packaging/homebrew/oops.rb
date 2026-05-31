class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.6.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.6.1/oops_darwin_arm64.tar.gz"
      sha256 "bdc7a5b03a416617f146241be7bf4810f3704ed99727cb614ff7a35a96a30a38"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.6.1/oops_darwin_amd64.tar.gz"
      sha256 "6b6c400b41fb02b112f051bc9df31867d624b3a34cf37d1832bd94096bd9b27a"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.6.1/oops_linux_arm64.tar.gz"
      sha256 "334f3f6deb96243b47067ddff404fbaf66912713a832db7ce15afa958ded91be"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.6.1/oops_linux_amd64.tar.gz"
      sha256 "afe16f757e4a9a6b8e1c555216778b4609a9ed6971bf3bbdfccdb99f442bf5c4"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
