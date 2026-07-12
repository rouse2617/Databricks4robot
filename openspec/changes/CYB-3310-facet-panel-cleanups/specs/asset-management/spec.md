# asset-management spec delta — CYB-3310

## MODIFIED — facet 面板展开管理

- **Given** 用户在 `/assets` 展开了一个或多个 facet 分组
- **When** 点击「全部收起」
- **Then** 所有分组收起(`expandedGroups` 清空),数据列表回到首屏可见;筛选条件与结果不受影响

- **Given** 多个 facet 分组同时展开、内容较高
- **When** 渲染 facet 面板
- **Then** 面板自身限高并内部滚动,不得把下方结果列表推出首屏

## MODIFIED — facet 文案可读性

- **Given** 场景标签输入框
- **Then** 占位符为面向用户的说明(示例值),不得暴露后端实现术语(如 "nested")

- **Given** 某枚举 facet(如环境)存在"空/未标注"取值(bucket key 为 `-` 或空串)
- **When** 渲染该选项
- **Then** 显示为「(未标注)」等可读文案;传给后端的筛选原始值保持不变
