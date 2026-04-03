# cc-download

[中文版 README](README_CN.md)

A CI/CD pipeline that automatically downloads official Claude Code binaries and publishes them to Alibaba Cloud OSS and GitHub Releases.

## Overview

cc-download is a GitHub Actions-based automated release pipeline. Whenever a new version tag is pushed, it fetches platform binaries from the official Claude Code GCS bucket, verifies SHA-256 integrity, uploads files to Alibaba Cloud OSS, and creates a GitHub Release.

A `latest.txt` file is always maintained on OSS at a fixed path, recording the current latest version number for downstream scripts to query automatically.

## Features

- **Multi-platform support**: Covers 6 platforms:
  - `darwin-arm64` — macOS Apple Silicon
  - `darwin-x64` — macOS Intel
  - `linux-arm64` — Linux ARM64 (glibc)
  - `linux-x64` — Linux x86_64 (glibc)
  - `linux-arm64-musl` — Linux ARM64 (musl/Alpine)
  - `linux-x64-musl` — Linux x86_64 (musl/Alpine)

- **Integrity verification**: Reads SHA-256 digests from the official `manifest.json` and verifies each binary after download; aborts immediately on mismatch.

- **OSS publishing**: Uploads the manifest, platform binaries, and SHA-256 checksum file to Alibaba Cloud OSS with the following path structure:
  ```
  claude-code-releases/
  ├── latest.txt                          # Latest version number (fixed path, updated on every release)
  └── {VERSION}/
      ├── manifest.json
      ├── checksums-sha256.txt
      ├── darwin-arm64/claude
      ├── darwin-x64/claude
      ├── linux-arm64/claude
      ├── linux-x64/claude
      ├── linux-arm64-musl/claude
      └── linux-x64-musl/claude
  ```

- **GitHub Release**: Automatically creates a Release with platform descriptions and usage instructions; attachments include all platform binaries and the checksum file.

## Usage

### Triggering a Release

Push any tag to the repository to trigger the pipeline:

```bash
git tag v1.2.3
git push origin v1.2.3
```

The tag name becomes the GitHub Release name. The version number (with the `v` prefix stripped) is used to fetch the corresponding binaries from upstream.

### Fetching the Latest Version

```bash
# Query the current latest version
VERSION=$(curl -fsSL https://<oss-domain>/claude-code-releases/latest.txt)

# Download the binary for your platform (linux-x64 example)
curl -fsSL "https://<oss-domain>/claude-code-releases/${VERSION}/linux-x64/claude" -o claude
chmod +x claude
```

### Required Secrets

Configure the following secrets under repository Settings → Secrets and variables → Actions:

| Secret | Description |
|---|---|
| `OSS_ACCESS_KEY_ID` | Alibaba Cloud RAM user Access Key ID |
| `OSS_ACCESS_KEY_SECRET` | Alibaba Cloud RAM user Access Key Secret |
| `OSS_BUCKET` | OSS bucket name |
| `OSS_ENDPOINT` | OSS endpoint (e.g. `oss-cn-hangzhou.aliyuncs.com`) |
