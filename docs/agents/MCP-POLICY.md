# MCP Policy

## Required MCP capabilities

| Capability | Purpose | Credential needed |
|------------|---------|-------------------|
| Linear | Issue creation, update, query, comments | LINEAR_API_KEY |

## Required when the change touches `Frontend/` code

| Capability | Purpose | When |
|------------|---------|------|
| Chrome DevTools | Navigate dev UI, screenshots, console/network | **Only if** the PR/commit diff includes files under `Frontend/` → after frontend dev deploy, before commit — see [`deploy-verification.md`](deploy-verification.md) §1.3 |

**Not required** for backend-only, sdk-only, docs-only, or infra-only PRs (even if you deploy backend dev).

### Chrome DevTools MCP — fallback（唯一规范，其它文档引用本节）

适用：**diff 含 `Frontend/`** 且已完成 frontend dev deploy，准备做 deploy 验收时。

| 顺序 | 条件 | Agent 必须做 |
|------|------|----------------|
| 1 | MCP 可用（`mcps/user-chrome-devtools/tools/` 可读且调用成功） | 按 [`deploy-verification.md`](deploy-verification.md) §1.3 + §6.1 自行导航、截图、读 console |
| 2 | MCP 不可用（未安装、IAP/登录阻塞、dev 不可达、工具报错） | 在 `decisions.md` 或 PR **Agent decisions** 记录：**原因 + 已完成的自动化**（`npm run test/build`、后端 smoke、HTTP 200、revision） |
| 3 | 步骤 2 | 请求用户 **补测** UI，并给出 **逐步操作清单**（来自 `tasks.md`）— **不是**默认「请用户随便点一下」 |
| 4 | 步骤 2 | **禁止**在未记录原因时声称 Frontend deploy 验收完成 |
| 5 | 始终 | **不得**用「用户手动点一遍」替代 MCP，当 MCP 实际可用且 dev 可达 |

**Backend-only PR：** 跳过 Chrome DevTools；用 curl / `api-guide-smoke.sh` 即可（见 deploy-verification §二、§6.2）。

## Rules for all MCP calls

1. **Read the tool schema before `CallMcpTool`** (mandatory — see below)
2. High-risk operations (delete, bulk modify, external side effects) require explicit user confirmation before execution
3. On authentication failure: attempt auth once, do not retry repeatedly
4. Never hardcode credentials in repository files
5. MCP servers not listed above are forbidden unless the user explicitly enables them

### How to read MCP tool schemas (Cursor)

Descriptors live under the Cursor project MCP cache (not in the git repo):

```bash
# List servers
ls "$HOME/.cursor/projects/<workspace-slug>/mcps/"

# List tools for Linear
ls "$HOME/.cursor/projects/<workspace-slug>/mcps/user-linear/tools/"

# Read one tool's parameters before calling
cat "$HOME/.cursor/projects/<workspace-slug>/mcps/user-linear/tools/get_issue.json"
```

Replace `<workspace-slug>` with the folder name for this repo (e.g. `Users-rick-Databricks4robot`).

**Agent workflow:**

1. Decide server + tool name (e.g. `user-linear` + `get_issue`).
2. Open `mcps/<server>/tools/<toolName>.json` — note `required` fields and types.
3. Call MCP with arguments matching the schema (real newlines in strings, not `\n` escapes).
4. If the call fails with validation error, re-read the JSON and fix args — do not spam retries.

**Example (Linear `get_issue`):**

```json
{ "arguments": { "type": "object", "properties": { "id": { "type": "string" } }, "required": ["id"] } }
```

```text
CallMcpTool server=user-linear toolName=get_issue arguments={"id": "CYB-978"}
```

**Chrome DevTools MCP:** tools under `mcps/user-chrome-devtools/tools/` (e.g. `navigate_page`, `take_snapshot`, `click`, `take_screenshot`, `list_console_messages`). Use only on **dev** frontend URLs per [`deploy-verification.md`](deploy-verification.md).

## High-risk operations (require user confirmation)

- Any `delete_*` operation
- Bulk state changes (e.g., closing multiple issues at once)
- Operations with external side effects (sending notifications, webhooks)
- Writing to production-facing systems

## Credential handling

- If a required credential is missing, prompt the user: "Please set {CREDENTIAL_NAME} as an environment variable"
- Do not guess or fabricate credentials
- Do not store credentials in any tracked file

## Adding new MCP servers

If the project needs a new MCP capability:
1. Propose it in the OpenSpec change (design.md)
2. After approval, add it to this policy file
3. Document the credential requirement
