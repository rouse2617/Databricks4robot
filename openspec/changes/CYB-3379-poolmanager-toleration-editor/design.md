# Design — CYB-3379 PoolManager toleration editor

## 数据流(纯读 → 增删改)

**改前**:
```
UI(PoolManager)
  ├── list: listExecutionTargets() ✅ 已有
  ├── create: 空按钮,无 handler ❌
  ├── update: 空按钮,无 handler ❌
  └── delete: 空按钮,无 handler ❌
```

**改后**:
```
UI(PoolManager)
  ├── list:   listExecutionTargets()        ← 已有
  ├── create: createExecutionTarget(body)   ← 新增,POST /api/v1/execution-targets
  ├── update: updateExecutionTarget(id, body)  ← 新增,PUT /:id
  └── delete: deleteExecutionTarget(id)     ← 新增,DELETE /:id
```

## 表单编辑器结构

Modal 表单当前推测只有 name/namespace 基础字段。新加两个复合字段:

### templateTolerations(list of objects)

用 antd `<Form.List name={["resourceDefaults", "templateTolerations"]}>`:

```tsx
<Form.List name={["resourceDefaults", "templateTolerations"]}>
  {(fields, { add, remove }) => (
    <>
      {fields.map((field) => (
        <Space key={field.key} align="baseline">
          <Form.Item name={[field.name, "key"]} rules={[{ required: true }]}>
            <Input placeholder="key (e.g. compute-tier)" />
          </Form.Item>
          <Form.Item name={[field.name, "operator"]} initialValue="Equal">
            <Select options={[{value:"Equal"},{value:"Exists"}]} />
          </Form.Item>
          <Form.Item
            name={[field.name, "value"]}
            dependencies={[[field.name, "operator"]]}
            rules={[
              ({ getFieldValue }) => ({
                validator(_, val) {
                  const op = getFieldValue(["resourceDefaults","templateTolerations", field.name, "operator"]);
                  if (op === "Exists" || (val && val.trim())) return Promise.resolve();
                  return Promise.reject(new Error("value 必填(operator=Equal 时)"));
                }
              })
            ]}
          >
            <Input placeholder="value (e.g. med)" />
          </Form.Item>
          <Form.Item name={[field.name, "effect"]} initialValue="NoSchedule">
            <Select options={[{value:"NoSchedule"},{value:"NoExecute"},{value:"PreferNoSchedule"}]} />
          </Form.Item>
          <DeleteOutlined onClick={() => remove(field.name)} />
        </Space>
      ))}
      <Button type="dashed" onClick={() => add({operator:"Equal", effect:"NoSchedule"})} icon={<PlusOutlined />}>
        添加 toleration
      </Button>
    </>
  )}
</Form.List>
```

### templateNodeSelector(map,同 pattern)

同上,只是每项 2 field(key/value),不需要 operator/effect。

## API client 三个函数

**`updateExecutionTarget`**:
```ts
export function updateExecutionTarget(
  id: string,
  body: Partial<ExecutionTarget>
): Promise<ExecutionTarget> {
  return request<ExecutionTarget>(
    "PUT",
    `/execution-targets/${encodeURIComponent(id)}`,
    body
  );
}
```

其他两个同样一行 delegate 到 `request()` helper(pipelineApi.ts 现有的 fetch 抽象)。

## 提交/取消 handler

Modal 的 `onOk` 里根据 `editTarget` state 判断新建 vs 编辑:

```tsx
const onOk = async () => {
  const values = await form.validateFields();
  try {
    if (editTarget) {
      await updateExecutionTarget(editTarget.id, values);
      message.success("已更新");
    } else {
      await createExecutionTarget(values);
      message.success("已创建");
    }
    setModalOpen(false);
    await fetchTargets();
  } catch (err) {
    message.error(err?.message || "保存失败");
    // 不关闭 modal,保留用户输入
  }
};
```

## 边界

- **`operator=Exists` 时 `value` 允许空** —— 特殊校验规则(见上)
- **删除**:antd `Modal.confirm` 二次确认,避免误删 target 导致所有走该 target 的 pipeline 崩
- **isDefault=true 的 target**:UI 层禁止删除按钮(避免删掉默认 target 挂全站)
- **PUT 幂等**:后端已提供,重复保存无副作用

## 风险 / 回滚

- 风险低:纯前端,后端 API 已生产上跑通
- 回滚:revert 单 commit,restore 原 PoolManager.tsx / pipelineApi.ts
- CI hard gates:openspec-gate / commitlint / pre-commit / prod frontend build,均现有

## 不做

- 不做通用「资源门槛 → toleration set」智能匹配
- 不做 K8s API poll / 集群感知
- 不改后端(zero backend delta)
