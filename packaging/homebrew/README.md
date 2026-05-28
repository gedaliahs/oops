# Homebrew tap

`oops.rb` is the formula for a future public tap:

```bash
brew install gedaliahs/tap/oops
```

Release flow:

1. Run `scripts/build-release.sh <version>`.
2. Run `scripts/update-homebrew-formula.sh <version>`.
3. Copy `packaging/homebrew/oops.rb` into `gedaliahs/homebrew-tap/Formula/oops.rb`.
4. Push the tap after the GitHub Release assets are live.

The formula installs the prebuilt GitHub Release archive for macOS/Linux and verifies the archive checksum.
