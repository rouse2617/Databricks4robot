# Linear Progress Templates

## Sub-Issue Detailed Update (full record)

```md
## 进度更新—<YYYY-MM-DD>：<本次交付标题>

拆出子 issue **<SUB-ISSUE-ID>（已 Done）** 记录本轮完整交付：
<SUB-ISSUE-URL>

### 交付要点

- <改动点 1>
- <改动点 2>
- <改动点 3>

### 实测（<场景名>）

| 指标 | 优化前 | 优化后 |
|---|---:|---:|
| <指标1> | <before> | <after> |
| 提升幅度 | - | <Nx> |

### 验证结果

- <功能一致性结论>
- <测试结论，如 go test ./... 通过>
- <环境验证结论，如 dev 已部署验证通过>

### 相关 commit

- branch: `<BRANCH>`
- commit: `<COMMIT_HASH>`
```

---

## Parent-Issue Brief Update (summary only)

```md
进度简报：<一句话状态结论>。

- 关键结果：<核心数字，如 13.2s -> 0.7s（18.7x）>
- 详细记录：<SUB-ISSUE-ID>（<SUB-ISSUE-URL>）
- 相关 commit：`<COMMIT_HASH>`（branch: `<BRANCH>`）
```

---

## Minimal plain version (when user asks "简要说明")

```md
- 问题：<一句话问题>
- 动作：<一句话动作>
- 结果：<一句话量化结果>
- 验证：<一句话验证结论>
- 状态：<是否阻塞上线>
```
