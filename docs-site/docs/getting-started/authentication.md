# Authentication

Cyber Databrew API 使用 Token 进行身份认证。所有 API 请求需要在 HTTP Header 中携带 `X-Databrew-Token`。

## API Token 认证

### Header 格式

```
X-Databrew-Token: your-token-here
```

### 示例

```bash
curl -H "X-Databrew-Token: g-xxx" \
  https://api-cyber-databrew.cyberorigin.ai/api/v1/assets
```

## SDK 认证方式

Cyber Databrew Python SDK 支持多种认证配置方式，按优先级从高到低排列：

### 1. 构造参数（最高优先级）

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(
    base_url="https://api-cyber-databrew.cyberorigin.ai",
    token="g-xxx",
)
```

### 2. 环境变量

```bash
export CYBER_DATABREW_BASE_URL="https://api-cyber-databrew.cyberorigin.ai"
export CYBER_DATABREW_TOKEN="g-xxx"
export CYBER_DATABREW_EMAIL="user@company.com"  # 仅用于审计
```

### 3. 配置文件

`~/.cyber-databrew/config.yaml`：

```yaml
base_url: https://api-cyber-databrew.cyberorigin.ai
default_token: g-xxx
default_email: user@company.com
timeout: 60
```

### 4. 远端获取

SDK 启动时自动调用 `GET /api/v1/sdk-config` 获取远端配置覆盖。后端不可用时静默回退。

### 5. 内置默认值

| 参数 | 默认值 |
|------|--------|
| `base_url` | `http://localhost:8080` |
| `timeout` | 30.0 秒 |

## 用户邮件头（审计用）

可选地在请求中添加 `X-User-Email` 头，用于审计追踪：

```python
client = CyberDatabrewClient(
    base_url="https://api-cyber-databrew.cyberorigin.ai",
    token="g-xxx",
    email="user@company.com",
)
```

## 错误处理

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
