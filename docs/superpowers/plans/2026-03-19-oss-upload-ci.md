# OSS Upload CI Step Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a new step to `.github/workflows/release.yml` that uploads all release assets to Alibaba Cloud OSS after checksum generation and before the GitHub Release creation.

**Architecture:** A single new shell step is inserted between the existing "Generate checksums file" and "Create GitHub Release" steps. The step installs ossutil v1.7.19, configures it non-interactively via four GitHub Secrets, and uploads 8 files (manifest.json, checksums-sha256.txt, and 6 platform binaries) to OSS using per-file `ossutil cp` invocations. Binary files are renamed from `claude-{VERSION}-{PLATFORM}` to `claude` at the OSS destination, mirroring the GCS path structure.

**Tech Stack:** GitHub Actions (YAML), ossutil v1.7.19 (Alibaba Cloud CLI), Bash

---

## Files

| Action | File | Change |
|---|---|---|
| Modify | `.github/workflows/release.yml` | Insert new step between lines 82 and 83 |

No other files change. Go service and nginx config are out of scope.

---

## Prerequisites (human action required before implementation)

The following 4 GitHub Secrets must be set in the repository settings under **Settings → Secrets and variables → Actions** before the workflow will succeed:

| Secret | Example value | Notes |
|---|---|---|
| `OSS_ACCESS_KEY_ID` | `LTAI5t...` | RAM user access key ID |
| `OSS_ACCESS_KEY_SECRET` | `...` | RAM user access key secret |
| `OSS_BUCKET` | `my-bucket` | OSS bucket name |
| `OSS_ENDPOINT` | `oss-cn-hangzhou.aliyuncs.com` | Public endpoint only — not VPC-internal |

The RAM user needs `oss:PutObject` on `acs:oss:*:*:{BUCKET}/claude-code-releases/*`.

---

### Task 1: Add the OSS upload step to the workflow

**Files:**
- Modify: `.github/workflows/release.yml` (insert new step after line 82, before the `Create GitHub Release` step)

- [ ] **Step 1: Validate the current workflow YAML is well-formed**

  Run:
  ```bash
  python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))" && echo "YAML OK"
  ```
  Expected output: `YAML OK`

  This gives a baseline so you know any subsequent YAML error was introduced by your edit.

- [ ] **Step 2: Insert the new OSS upload step into the workflow**

  Open `.github/workflows/release.yml`. After the `Generate checksums file` step (which ends at line 81) and before the `Create GitHub Release` step (which starts at line 83), insert the following step:

  ```yaml
      - name: Upload to Alibaba Cloud OSS
        env:
          VERSION: ${{ steps.version.outputs.version }}
          OSS_ACCESS_KEY_ID: ${{ secrets.OSS_ACCESS_KEY_ID }}
          OSS_ACCESS_KEY_SECRET: ${{ secrets.OSS_ACCESS_KEY_SECRET }}
          OSS_BUCKET: ${{ secrets.OSS_BUCKET }}
          OSS_ENDPOINT: ${{ secrets.OSS_ENDPOINT }}
        run: |
          set -euo pipefail

          PLATFORMS=("darwin-arm64" "darwin-x64" "linux-arm64" "linux-x64" "linux-arm64-musl" "linux-x64-musl")

          echo "==> Installing ossutil v1.7.19..."
          curl -fsSL https://gosspublic.alicdn.com/ossutil/1.7.19/ossutil64 -o ossutil
          chmod +x ossutil

          echo "==> Configuring OSS credentials..."
          ./ossutil config -e "$OSS_ENDPOINT" -i "$OSS_ACCESS_KEY_ID" -k "$OSS_ACCESS_KEY_SECRET" -L CH

          echo "==> Uploading manifest.json..."
          ./ossutil cp "./release-assets/manifest.json" \
            "oss://${OSS_BUCKET}/claude-code-releases/${VERSION}/manifest.json" \
            --force --meta "Content-Type:application/json"

          echo "==> Uploading checksums-sha256.txt..."
          ./ossutil cp "./release-assets/checksums-sha256.txt" \
            "oss://${OSS_BUCKET}/claude-code-releases/${VERSION}/checksums-sha256.txt" \
            --force --meta "Content-Type:text/plain"

          echo "==> Uploading platform binaries..."
          for PLATFORM in "${PLATFORMS[@]}"; do
            echo "  Uploading $PLATFORM..."
            ./ossutil cp "./release-assets/claude-${VERSION}-${PLATFORM}" \
              "oss://${OSS_BUCKET}/claude-code-releases/${VERSION}/${PLATFORM}/claude" \
              --force --meta "Content-Type:application/octet-stream"
          done

          echo ""
          echo "==> OSS upload complete. Uploaded paths:"
          echo "  oss://${OSS_BUCKET}/claude-code-releases/${VERSION}/manifest.json"
          echo "  oss://${OSS_BUCKET}/claude-code-releases/${VERSION}/checksums-sha256.txt"
          for PLATFORM in "${PLATFORMS[@]}"; do
            echo "  oss://${OSS_BUCKET}/claude-code-releases/${VERSION}/${PLATFORM}/claude"
          done
  ```

  The final workflow step order must be:
  1. Checkout repository
  2. Get version from tag
  3. Download binaries and verify checksums
  4. Generate checksums file
  5. **Upload to Alibaba Cloud OSS** ← new step
  6. Create GitHub Release

- [ ] **Step 3: Validate the modified workflow YAML is still well-formed**

  Run:
  ```bash
  python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))" && echo "YAML OK"
  ```
  Expected output: `YAML OK`

  If you get a YAML parse error, check indentation — GitHub Actions steps must be indented with exactly 6 spaces (2 for `jobs`, 2 for `release`, 2 for `steps`). The `env:` and `run:` keys inside the step are at 8 spaces; the run script content is at 10 spaces.

- [ ] **Step 4: Verify the step order visually**

  Run:
  ```bash
  grep -n "^\s*- name:" .github/workflows/release.yml
  ```
  Expected output:
  ```
  16:      - name: Checkout repository
  20:      - name: Get version from tag
  27:      - name: Download binaries and verify checksums
  77:      - name: Generate checksums file
  83:      - name: Upload to Alibaba Cloud OSS
  NN:      - name: Create GitHub Release
  ```
  (Line numbers for the new step and "Create GitHub Release" will shift from original; confirm "Upload to Alibaba Cloud OSS" appears before "Create GitHub Release".)

- [ ] **Step 5: Commit**

  ```bash
  git add .github/workflows/release.yml
  git commit -m "feat: upload release assets to Alibaba Cloud OSS in CI"
  ```

---

### Task 2: Verify after first real run

This task is performed manually after a real tag push triggers the workflow.

- [ ] **Step 1: Confirm the workflow run succeeded**

  In GitHub Actions, verify the "Upload to Alibaba Cloud OSS" step shows green with output like:
  ```
  ==> OSS upload complete. Uploaded paths:
    oss://my-bucket/claude-code-releases/X.Y.Z/manifest.json
    oss://my-bucket/claude-code-releases/X.Y.Z/checksums-sha256.txt
    oss://my-bucket/claude-code-releases/X.Y.Z/darwin-arm64/claude
    ...
  ```

- [ ] **Step 2: Spot-check a binary is publicly accessible**

  Replace `{BUCKET}`, `{ENDPOINT}`, and `{VERSION}` with real values:
  ```bash
  curl -I "https://{BUCKET}.{ENDPOINT}/claude-code-releases/{VERSION}/darwin-arm64/claude"
  ```
  Expected: `HTTP/1.1 200 OK`

- [ ] **Step 3: Confirm all 8 objects exist**

  ```bash
  ./ossutil ls "oss://{BUCKET}/claude-code-releases/{VERSION}/"
  ```
  Expected: 8 objects (6 platform `claude` binaries + `manifest.json` + `checksums-sha256.txt`).
