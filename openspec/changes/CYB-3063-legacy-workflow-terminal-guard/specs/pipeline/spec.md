## ADDED Requirements

### Requirement: 终态 run 查询 workflow 详情不再直连 Argo
The system SHALL avoid querying the live Argo API for a workflow's detail view when the corresponding run has already reached a terminal state and the underlying workflow object still exists.

**Priority**: P1 (High)
**Rationale**: 新的 Run Kernel 路径已经对终态 run 有防护,这条旧接口是唯一的漏网之鱼——每次打开详情页都会对一个早已跑完、状态不会再变化的 run 重复问一次 Argo,浪费 API 调用且增加集群负担。

#### Scenario: 已终态且底层对象仍存在的 run
- **Given** 一个 run 已经处于终态(成功/失败/错误),且对应的 Argo workflow 对象还没有被清理
- **When** 用户打开这个 run 的详情页(或直接调用对应接口)
- **Then** 返回的响应数据与原有直连 Argo 得到的数据在核心字段上一致,且这次请求没有触发对 Argo API 的实时调用

#### Scenario: 活跃(未终态)run 保持原有行为
- **Given** 一个 run 仍在执行中,还没有到达终态
- **When** 用户打开这个 run 的详情页
- **Then** 系统照常查询 Argo 拿实时状态,行为与改动前一致

### Requirement: 降级路径数据不完整时安全回退,不返回残缺响应
The system SHALL fall back to querying the live Argo API when the data needed to construct a degraded (non-Argo) response is missing or invalid, rather than returning a response with silently missing or incorrect fields.

**Priority**: P0 (Critical)
**Rationale**: 数据完整性优先于"减少 Argo 调用"这个优化目标——用残缺或错误的数据拼一个看起来正常的响应,比多打一次 Argo 的代价更高,会误导用户对 run 真实状态的判断。

#### Scenario: 构造降级响应所需的历史数据缺失
- **Given** 一个 run 已经终态,但用于构造降级响应的历史数据(如提交时的 workflow 定义快照)缺失或无法解析
- **When** 用户打开这个 run 的详情页
- **Then** 系统回退到直连 Argo 查询实时状态,不返回一个字段缺失或错误的响应
