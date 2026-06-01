# Authentication

Cyber Databrew 支持两类认证方式：SDK / 脚本使用 `X-Databrew-Token`，浏览器端使用邮箱登录换取 `databrew_session` JWT cookie。

## API Token Authentication

### Header

```
X-Databrew-Token: your-token-here
```

### Example

```bash
curl -H "X-Databrew-Token: g-xxx" \
  https://api-cyber-databrew.cyberorigin.ai/api/v1/assets
```

## Browser Email Login

浏览器端实际登录入口是 `POST /api/v1/auth/email-login`。登录成功后，后端设置 `databrew_session` JWT cookie，前端后续 API 请求依赖浏览器自动带上该 cookie。

```bash
curl -X POST http://localhost:8080/api/v1/auth/email-login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com"}' \
  -c cookies.txt
```

浏览器登录和 SDK token 是不同场景：不要把 `email-login` 当作 SDK token 获取接口，也不要要求浏览器端手动设置 `X-Databrew-Token`。

## SDK Authentication

Cyber Databrew Python SDK 支持多种认证配置方式，按优先级从高到低排列：

### 1. Constructor Arguments

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(
    base_url="https://api-cyber-databrew.cyberorigin.ai",
    token="g-xxx",
)
```

### 2. Environment Variables

```bash
export CYBER_DATABREW_BASE_URL="https://api-cyber-databrew.cyberorigin.ai"
export CYBER_DATABREW_TOKEN="g-xxx"
export CYBER_DATABREW_EMAIL="user@company.com"  # 仅用于审计
```

### 3. Config File

`~/.cyber-databrew/config.yaml`：

```yaml
base_url: https://api-cyber-databrew.cyberorigin.ai
default_token: g-xxx
default_email: user@company.com
timeout: 60
```

### 4. Remote Defaults

SDK 启动时自动调用 `GET /api/v1/sdk-config` 获取远端配置覆盖。后端不可用时静默回退。

### 5. Built-in Defaults

| 参数 | 默认值 |
|------|--------|
| `base_url` | `http://localhost:8080` |
| `timeout` | 30.0 秒 |

## User Email Header

可选地在请求中添加 `X-User-Email` 头，用于审计追踪：

```python
client = CyberDatabrewClient(
    base_url="https://api-cyber-databrew.cyberorigin.ai",
    token="g-xxx",
    email="user@company.com",
)
```

## Error Handling

```python
from cyber_databrew_sdk import CyberDatabrewClient
from cyber_databrew_sdk.exceptions import AuthenticationError, RateLimitError

client = CyberDatabrewClient(base_url="...", token="g-xxx")

try:
    asset = client.assets.get("some-id")
except AuthenticationError as e:
    print(f"鉴权失败，请检查 Token: {e}")
except RateLimitError as e:
    print(f"被限流，请稍后重试: {e}")
```
