class Oops < Formula
  desc "Terminal undo for destructive commands"
  homepage "https://oops-cli.com"
  version "0.6.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.6.0/oops_darwin_arm64.tar.gz"
      sha256 "0069b02cd759d839893b025298df2535291fc9286d61e5fbc212f926da949a30"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.6.0/oops_darwin_amd64.tar.gz"
      sha256 "bc09d30927cde6d4050adab49f7bdd206bddf8d27a232afe55b9d52949796acb"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gedaliahs/oops/releases/download/v0.6.0/oops_linux_arm64.tar.gz"
      sha256 "acce3dd8b643d5ce1fc4ae0e7e340311306d845a49ee708dcdaa2944369298ae"
    else
      url "https://github.com/gedaliahs/oops/releases/download/v0.6.0/oops_linux_amd64.tar.gz"
      sha256 "40cd83d715cf3dda376ed37b7ee6cc35e8e043795fb18fd8622d831f65a6dbcf"
    end
  end

  def install
    bin.install "oops"
  end

  test do
    assert_match "oops v#{version}", shell_output("#{bin}/oops --version")
  end
end
