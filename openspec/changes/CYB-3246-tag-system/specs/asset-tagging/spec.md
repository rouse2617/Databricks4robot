# Spec Delta — asset-tagging (CYB-3246)

## MODIFIED Requirements

### Requirement: 标签写入校验（开放词汇）
The system SHALL accept a tag whose key is not present in the tag registry by
treating it as a free-form string value (subject to a default maximum length),
while continuing to validate any registered key strictly against its definition
(enum membership or string length).

- **Before**: The system SHALL reject any tag whose key is not registered in the tag registry.
- **After**: The system SHALL accept an unregistered key as a free-form string tag, and SHALL still reject registered keys whose value violates their definition.
- **Reason**: 用户需在不改配置文件、不重启服务的前提下为资产打任意语义标签；受管标签的 enum/长度语义必须保持不回退。

**Priority**: P0 (Critical)
**Rationale**: 未注册即拒绝是当前无法自助打标签的根因；标签又是全文检索的主要内容载体。

#### Scenario: 未注册 string key 被接受
- **Given** 标签注册表中不存在 key `description`
- **When** 用户为某资产写入标签 `description = "高清城市街道场景"`
- **Then** 写入成功，返回该资产且其标签集合包含 `description`，其值可被后续读取与全文检索命中

#### Scenario: 已注册 enum 非法值仍被拒绝
- **Given** 注册表中 `priority` 为 enum，允许值为 `critical/high/medium/low`
- **When** 用户写入 `priority = "urgent"`
- **Then** 写入被拒绝并返回校验错误（HTTP 422），资产标签不变

#### Scenario: 未注册 key 超默认长度被拒绝
- **Given** 未注册 key 的默认最大长度为 500 字符
- **When** 用户写入一个长度超过 500 字符的未注册标签值
- **Then** 写入被拒绝并返回校验错误（HTTP 422）

## ADDED Requirements

### Requirement: 标签展示可读性
The system SHALL present each asset tag such that overly long values are
truncated with an affordance to view the full value, the tag source is shown
distinctly, and deletion requires explicit confirmation.

**Priority**: P1 (High)
**Rationale**: 当前标签被截断、来源混排、误删无提示，直接影响可读性与安全性。

#### Scenario: 超长标签值可展开查看
- **Given** 某资产的标签值长度超过展示阈值
- **When** 用户在资产详情页查看该标签
- **Then** 值以省略形式展示，并提供展开查看完整内容的交互

#### Scenario: 删除标签需确认
- **Given** 用户在资产详情页看到一个已存在的标签
- **When** 用户点击删除该标签
- **Then** 系统先要求确认，确认后才移除该标签

### Requirement: 标签注册管理
The system SHALL allow an authorized administrator to create, update, and
remove managed tag definitions (including enum keys and their allowed values)
through an interface, and newly registered definitions SHALL take effect for
subsequent tag writes without a service restart.

**Priority**: P1 (High)
**Rationale**: 消除「改 YAML + 重启」链路，让受管标签可自助治理。

#### Scenario: 新注册的 enum 标签即时生效
- **Given** 管理员在标签管理界面注册 enum 标签 `severity`，允许值为 `critical/high/low`
- **When** 随后有资产写入 `severity = "high"`（服务未重启）
- **Then** 该写入通过校验并成功；写入 `severity = "fatal"` 则被拒绝

#### Scenario: 非管理员无权管理注册表
- **Given** 一个不具备管理员身份的调用者
- **When** 其尝试创建或删除标签定义
- **Then** 请求被拒绝（HTTP 401/403），注册表不变

#### Scenario: 重复 key 注册被拒绝
- **Given** 注册表中已存在 key `severity`
- **When** 管理员再次以相同 key `severity` 创建定义
- **Then** 请求被拒绝并返回冲突错误，原定义不被覆盖
