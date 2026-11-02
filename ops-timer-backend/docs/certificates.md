# 证书管理与附件

在侧栏「证书管理」上传证书，或从项目、计时单元详情的「证书附件」上传。支持单个文件中包含多张证书；每张证书分别显示元数据和到期时间，原始文件作为一个附件保存。

## 格式与信息

- X.509：PEM（`CERTIFICATE`，支持证书链）及 DER，提取 CN、主体、签发者、序列号、SHA-256 指纹、公钥算法、DNS 名称、邮箱、生效和到期时间。
- GPG / OpenPGP：ASCII armor 公钥块或二进制公钥，支持现代 ECC 和 v6 公钥，提取名称、用户 ID、邮箱、Key ID、公钥指纹、算法、创建时间、主公钥有效期及撤销标记。GPG 没有 X.509 的 CN 字段，名称来自主身份的用户 ID。
- 不解析、解密或存储私钥。PEM 私钥块、OpenPGP Secret Key / Secret Subkey 包、私钥与证书混合的文件均拒绝。PKCS#12（PFX / P12）、CSR 和普通裸公钥不属于支持格式。
- 单文件上限 4 MiB，按文件内容识别格式。附件列表只返回元数据，下载接口返回上传的原始字节。
- 页面根据当前时间展示剩余时长、30 天内到期、已过期、尚未生效或已撤销。GPG 未声明有效期时显示「永不过期」，不能创建到期倒计时。GPG 的到期时间以主公钥自签名为准，不合并各子密钥的有效期。
- 这是元数据解析功能，不验证 X.509 信任链，不查询在线吊销状态；GPG 撤销标记仅来自文件内的有效撤销签名。

## 倒计时与关联

点击任意有到期时间的证书的「到期倒计时」，以其准确到期时间创建时间倒计时，沿用现有仪表盘和通知扫描，默认提前 30、7、1 天提醒。过期证书也可以创建倒计时以展示超期状态；已撤销的 GPG 证书不能创建倒计时。

同一证书重复点击返回已有倒计时；若倒计时已删除，可以重新创建。原证书自动关联到新倒计时；从项目上传的证书创建倒计时时归属该项目。倒计时创建后仍可按原有功能编辑。

项目、计时单元可关联已有证书，也可移除关联。移除关联保留原证书；在证书管理页删除附件会删除证书文件、解析信息和全部关联，但保留已有倒计时。

附件关联 API 还支持 `todo`、`note`、`schedule` 对象，可供其他客户端接入。

## REST API

以下路径均在 `/api/v1` 下，采用系统现有登录认证。OAuth 使用 `data:read` / `data:write`。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/attachments` | `multipart/form-data` 上传，必填 `file`；可同时传 `owner_type`、`owner_id` |
| GET | `/attachments` | 分页列表：`q` 搜索文件名/CN/主体/指纹，`type=x509\|gpg`，`owner_type` 与 `owner_id` 成对使用，`page`，`page_size`（1–100） |
| GET | `/attachments/:id/download` | 下载原始证书文件 |
| DELETE | `/attachments/:id` | 删除附件及证书、全部关联 |
| POST | `/attachments/:id/links` | JSON `{ "owner_type": "project", "owner_id": "UUID" }`，幂等关联 |
| DELETE | `/attachments/:id/links` | 相同 JSON，移除指定关联 |
| POST | `/certificates/:id/countdown` | 为单张证书创建或返回已有倒计时；响应 `data.id` 是计时单元 ID |

附件响应包含 `id`、`filename`、`content_type`、`size`、`created_at`、`certificates[]`。证书信息中的 `expires_at` 为 RFC3339 时间或 `null`，`remaining_seconds` 为剩余秒数（过期为负）或 `null`，`countdown_unit_id` 记录最后生成的倒计时 ID。

## 存储与升级

原始附件以 PostgreSQL `bytea` 保存；元数据和关联分别存入 `certificates`、`attachment_links` 表。启动时自动迁移，无需额外文件目录或卷。

备份格式升级到 schema version 2，包含附件原始字节（JSON Base64）、证书元数据和关联。仍支持旧版备份；恢复时重新解析证书，拒绝含私钥的附件，并从原文件重新生成到期等元数据。旧版后端不能恢复 version 2 备份，因此升级时需同步更新后端和前端。

## 验证

解析测试涵盖 X.509 PEM / DER / 证书链、GPG v4 / v6、有效期和无有效期、公私钥混合拒绝、私钥子包拒绝、尾随垃圾及损坏文件。

```sh
cd ops-timer-backend
go test ./internal/pkg/certparse
# 测试使用独立 schema，不清理现有业务表。
TASK_MANAGER_TEST_DATABASE_DSN='postgres://user:password@localhost/test?sslmode=disable' go test ./...
```

`ops-timer-frontend/e2e/certificates.spec.ts` 覆盖浏览器上传、元数据、原文件下载、倒计时、拒绝私钥、移除/重新关联和删除。运行方式沿用其他 Playwright 测试，需有独立测试后端和管理员账户。
