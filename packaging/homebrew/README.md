# Homebrew tap

`oops.rb` is the formula for the public tap:

```bash
brew install gedaliahs/tap/oops
```

Release flow:

1. Run `scripts/build-release.sh <version>`.
2. Run `scripts/update-homebrew-formula.sh <version>`.
3. Push a `v<version>` tag.
4. The release workflow uploads the assets and pushes the formula to `gedaliahs/homebrew-tap` when `HOMEBREW_TAP_TOKEN` is configured.

The formula installs the prebuilt GitHub Release archive for macOS/Linux and verifies the archive checksum.
