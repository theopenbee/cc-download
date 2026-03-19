# Design: Upload Release Binaries to Alibaba Cloud OSS in CI

**Date:** 2026-03-19
**Status:** Approved
**Scope:** `.github/workflows/release.yml` only — no changes to Go service or nginx config.

---

## Background

The existing CI workflow downloads Claude Code binaries from Google Cloud Storage (GCS), verifies checksums, and publishes a GitHub Release. This design adds a step to mirror all release assets to Alibaba Cloud OSS, providing a distribution point served from Alibaba Cloud infrastructure.

---

## Goals

- Upload all release assets to OSS after download and verification.
- Mirror the GCS path structure exactly so OSS can serve as a drop-in replacement source.
- Fail the CI job if any upload fails, preventing an incomplete mirror.

## Non-Goals

- Changes to the Go HTTP service (`main.go`).
- Changes to nginx configuration.
- Making the Go service read from OSS instead of local disk.
- Bucket creation or access-control configuration (done outside CI).

---

## OSS Path Structure

All files are placed under `claude-code-releases/{VERSION}/`, mirroring the GCS layout.

| Local file (in `./release-assets/`) | OSS destination |
|---|---|
| `manifest.json` | `claude-code-releases/{VERSION}/manifest.json` |
| `checksums-sha256.txt` | `claude-code-releases/{VERSION}/checksums-sha256.txt` |
| `claude-{VERSION}-darwin-arm64` | `claude-code-releases/{VERSION}/darwin-arm64/claude` |
| `claude-{VERSION}-darwin-x64` | `claude-code-releases/{VERSION}/darwin-x64/claude` |
| `claude-{VERSION}-linux-arm64` | `claude-code-releases/{VERSION}/linux-arm64/claude` |
| `claude-{VERSION}-linux-x64` | `claude-code-releases/{VERSION}/linux-x64/claude` |
| `claude-{VERSION}-linux-arm64-musl` | `claude-code-releases/{VERSION}/linux-arm64-musl/claude` |
| `claude-{VERSION}-linux-x64-musl` | `claude-code-releases/{VERSION}/linux-x64-musl/claude` |

The bucket is configured as **public read**, so files are accessible via:
`https://{BUCKET}.{ENDPOINT}/claude-code-releases/{VERSION}/{PLATFORM}/claude`

---

## Implementation Approach: Direct ossutil Per-File Upload

Use `ossutil` v1.x (official Alibaba Cloud CLI, single binary) to upload each file explicitly.

**Why this approach:**
- Full control over source-to-destination path mapping.
- Matches the explicit, readable style of the existing download step.
- No risk of unintended deletions (unlike `ossutil sync`).
- Simple error handling: `set -euo pipefail` stops on first failure.

**Rejected alternatives:**
- *Stage-then-sync*: Extra staging step, `sync` may delete older versions on the bucket.
- *Parallel background uploads (`&` + `wait`)*: Complicates error handling for marginal speed gain.

---

## New CI Step

**Position:** After "Generate checksums file", before "Create GitHub Release".
**Rationale:** If OSS upload fails, the job stops before creating the GitHub Release, avoiding a partially-published release that lacks its OSS mirror.

### Step logic

1. Download `ossutil64` binary from the official Alibaba CDN and make it executable.
2. Write OSS credentials to `~/.ossutilconfig` using `ossutil config` (credentials come from GitHub Secrets and are not echoed to logs).
3. Upload `manifest.json` with `Content-Type: application/json`.
4. Upload `checksums-sha256.txt` with `Content-Type: text/plain`.
5. Loop over all 6 platforms; upload each binary with `Content-Type: application/octet-stream` to its platform subdirectory path.
6. Print a summary of uploaded OSS paths.

All uploads use `--force` to allow idempotent re-runs.

---

## Required GitHub Secrets

| Secret name | Example value | Description |
|---|---|---|
| `OSS_ACCESS_KEY_ID` | `LTAI5t...` | RAM user access key ID |
| `OSS_ACCESS_KEY_SECRET` | `...` | RAM user access key secret |
| `OSS_BUCKET` | `my-bucket` | OSS bucket name |
| `OSS_ENDPOINT` | `oss-cn-hangzhou.aliyuncs.com` | OSS regional endpoint |

The RAM user needs `oss:PutObject` permission on `acs:oss:*:*:{BUCKET}/claude-code-releases/*`.

---

## Error Handling

- The step runs under `set -euo pipefail`; any `ossutil cp` failure stops the job immediately.
- ossutil exits non-zero on upload failure, upload timeout, or authentication error.
- The GitHub Release step is only reached if all OSS uploads succeed.

---

## Security Considerations

- Credentials are stored as GitHub encrypted secrets and injected via `env:` — never hardcoded.
- `ossutil config` writes credentials to `~/.ossutilconfig` on the ephemeral runner; the file is discarded when the runner terminates.
- The bucket is public read but write access requires the RAM credentials above.

---

## Testing / Verification

After the first successful run, verify by curling a file directly from OSS:

```bash
curl -I "https://{BUCKET}.{ENDPOINT}/claude-code-releases/{VERSION}/darwin-arm64/claude"
# Expected: HTTP/1.1 200 OK
```

And confirm all 8 paths exist using:

```bash
ossutil ls oss://{BUCKET}/claude-code-releases/{VERSION}/
```
