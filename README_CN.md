# cc-download

自动化下载 Claude Code 官方二进制文件并发布到阿里云 OSS 与 GitHub Releases 的 CI/CD 工程。

## 项目介绍

cc-download 是一个基于 GitHub Actions 的自动化发布流水线。每当推送新版本 tag 时，它会从 Claude Code 官方 GCS 存储桶拉取各平台二进制文件、校验 SHA-256 完整性，然后将文件同步上传至阿里云 OSS，并同步创建 GitHub Release。

OSS 上始终维护一个 `latest.txt` 文件，记录当前最新版本号，便于下游脚本自动获取最新版本。

## 功能说明

- **多平台支持**：覆盖以下 6 个平台：
  - `darwin-arm64` — macOS Apple Silicon
  - `darwin-x64` — macOS Intel
  - `linux-arm64` — Linux ARM64 (glibc)
  - `linux-x64` — Linux x86_64 (glibc)
  - `linux-arm64-musl` — Linux ARM64 (musl/Alpine)
  - `linux-x64-musl` — Linux x86_64 (musl/Alpine)

- **完整性校验**：从官方 `manifest.json` 读取 SHA-256 摘要，下载后逐一比对，不匹配则立即中止。

- **OSS 发布**：将 manifest、各平台二进制及 SHA-256 校验文件上传至阿里云 OSS，路径结构如下：
  ```
  claude-code-releases/
  ├── latest.txt                          # 最新版本号（固定路径，每次发布自动更新）
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

- **GitHub Release**：自动创建带有平台说明和使用说明的 Release，附件包含所有平台二进制和校验文件。

## 使用方式

### 触发发布

向仓库推送任意 tag 即可触发流水线：

```bash
git tag v1.2.3
git push origin v1.2.3
```

tag 名称即为 GitHub Release 名称，去掉前缀 `v` 后作为 Claude Code 版本号用于从上游拉取文件。

### 获取最新版本

```bash
# 查询当前最新版本号
VERSION=$(curl -fsSL https://<oss-domain>/claude-code-releases/latest.txt)

# 下载对应平台二进制（以 linux-x64 为例）
curl -fsSL "https://<oss-domain>/claude-code-releases/${VERSION}/linux-x64/claude" -o claude
chmod +x claude
```

### 所需 Secrets

在仓库 Settings → Secrets and variables → Actions 中配置以下 secrets：

| Secret 名称 | 说明 |
|---|---|
| `OSS_ACCESS_KEY_ID` | 阿里云 RAM 用户 Access Key ID |
| `OSS_ACCESS_KEY_SECRET` | 阿里云 RAM 用户 Access Key Secret |
| `OSS_BUCKET` | OSS Bucket 名称 |
| `OSS_ENDPOINT` | OSS Endpoint（如 `oss-cn-hangzhou.aliyuncs.com`）|
