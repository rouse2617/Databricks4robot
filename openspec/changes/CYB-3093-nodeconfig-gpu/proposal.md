# Proposal — CYB-3093

## Why
画布节点的「配置节点」覆盖面板(`NodeConfigPanel`)支持覆盖 CPU / 内存 / 磁盘,但缺 GPU 和计算档位,与「组件定义」表单不一致——用户无法在画布上给某个节点单独覆盖 GPU,只能回到组件定义里改。后端 transpiler 与 DSL 序列化都已支持节点级 GPU,唯独 UI 缺入口。

## What Changes

### Modified Capabilities
- 画布节点配置覆盖面板 SHALL 提供 GPU(数量)与计算档位两个字段,与 CPU/内存/磁盘 并列;打开时从节点数据回显,保存时写回节点数据。

## Impact
- **Affected code**: `Frontend/src/components/pipeline/NodeConfigPanel.tsx`(+ 其测试)
- **New APIs**: 无
- **Dependencies**: 无
- **Schema**: 无迁移(后端 `transpiler` 已读节点 `gpu`;前端 `canvas-to-dsl` 已序列化 `gpu`/`computeTier`)

## Scope
- **In scope**: `NodeConfigPanel` 补 GPU + 计算档位字段(表单值类型、初值回显、保存回写、UI 输入框)。
- **Out of scope**: 后端改动;组件定义表单(已有 GPU);端口移除持久化(CYB-3094)、onExit 节点在 DAG 露出(CYB-3095)另行处理。

## Success Criteria
- [ ] 「配置节点」面板出现 GPU 与计算档位输入框。
- [ ] 已有节点数据里的 gpu/computeTier 能回显;编辑后保存能写回并再次打开时保留。
- [ ] 不影响 CPU/内存/磁盘 覆盖与其他既有行为。
