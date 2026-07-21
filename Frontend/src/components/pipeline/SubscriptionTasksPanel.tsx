import {
  DeleteOutlined,
  EditOutlined,
  MinusCircleOutlined,
  MoreOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  App,
  Button,
  Card,
  Divider,
  Drawer,
  Dropdown,
  Form,
  Input,
  InputNumber,
  message,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  type ExecutionTarget,
  listExecutionTargets,
  listPipelines,
  type PipelineTemplate,
} from "../../api/pipelineApi";
import {
  createSubscriptionTask,
  deleteSubscriptionTask,
  listSubscriptionTasks,
  pauseSubscriptionTask,
  resumeSubscriptionTask,
  type SubscriptionTask,
  type SubscriptionTaskCreateRequest,
  updateSubscriptionTask,
} from "../../api/subscriptionTaskApi";

const { Text } = Typography;

interface BindingFormValue {
  templateId: string;
  templateVersion?: number;
  targetId: string;
}

interface TaskFormValues {
  name: string;
  projectId: string;
  subscriptionId: string;
  pullIntervalSeconds?: number;
  maxMessagesPerPull?: number;
  pipelineBindings: BindingFormValue[];
}

const STATUS_MAP: Record<string, { color: string; label: string }> = {
  succeeded: { color: "green", label: "成功" },
  failed: { color: "red", label: "失败" },
  empty: { color: "default", label: "无消息" },
};

const DEFAULT_PROJECT_ID = "green-valley-442103";

export function SubscriptionTasksPanel() {
  const { modal } = App.useApp();
  const [tasks, setTasks] = useState<SubscriptionTask[]>([]);
  const [loading, setLoading] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<SubscriptionTask | null>(null);
  const [form] = Form.useForm<TaskFormValues>();

  const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
  const [targets, setTargets] = useState<ExecutionTarget[]>([]);

  const fetchTasks = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await listSubscriptionTasks({ pageSize: 200 });
      setTasks(resp.items ?? []);
    } catch {
      message.error("加载订阅任务失败");
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchPickerData = useCallback(async () => {
    try {
      const [tplResp, tgtResp] = await Promise.all([
        listPipelines({ pageSize: 200 }),
        listExecutionTargets(),
      ]);
      setTemplates(
        Array.isArray(tplResp) ? tplResp : (tplResp as any).items ?? [],
      );
      setTargets(Array.isArray(tgtResp) ? tgtResp : (tgtResp as any).items ?? []);
    } catch {
      /* best-effort */
    }
  }, []);

  useEffect(() => {
    fetchTasks();
    fetchPickerData();
  }, [fetchTasks, fetchPickerData]);

  const templateMap = useMemo(
    () => new Map(templates.map((t) => [t.id, t])),
    [templates],
  );
  const targetMap = useMemo(
    () => new Map(targets.map((t) => [t.id, t])),
    [targets],
  );

  const templateOptions = useMemo(
    () =>
      templates.map((t) => ({
        value: t.id,
        label: `${t.name} (v${t.version ?? "?"})`,
      })),
    [templates],
  );
  const targetOptions = useMemo(
    () => targets.map((t) => ({ value: t.id, label: t.name })),
    [targets],
  );

  const openCreate = () => {
    setEditingTask(null);
    form.resetFields();
    form.setFieldsValue({
      pullIntervalSeconds: 10,
      maxMessagesPerPull: 1000,
      projectId: DEFAULT_PROJECT_ID,
      pipelineBindings: [{ templateId: "", targetId: "" }],
    });
    setDrawerOpen(true);
  };

  const openEdit = (task: SubscriptionTask) => {
    setEditingTask(task);
    form.setFieldsValue({
      name: task.name,
      projectId: task.projectId,
      subscriptionId: task.subscriptionId,
      pullIntervalSeconds: task.pullIntervalSeconds,
      maxMessagesPerPull: task.maxMessagesPerPull,
      pipelineBindings: (task.pipelineBindings ?? []).map((b) => ({
        templateId: b.templateId,
        templateVersion: b.templateVersion ?? undefined,
        targetId: b.targetId,
      })),
    });
    setDrawerOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const vals = await form.validateFields();
      // Advanced knobs are defaulted, not surfaced in the form: project falls
      // back to Databrew's own GCP project, interval/maxMessages to sensible
      // defaults, and template version to latest. Editing preserves whatever an
      // existing task already had.
      const body: SubscriptionTaskCreateRequest = {
        name: vals.name,
        projectId: editingTask?.projectId ?? DEFAULT_PROJECT_ID,
        subscriptionId: vals.subscriptionId,
        pullIntervalSeconds: editingTask?.pullIntervalSeconds ?? 10,
        maxMessagesPerPull: editingTask?.maxMessagesPerPull ?? 1000,
        pipelineBindings: (vals.pipelineBindings ?? []).map((b) => ({
          templateId: b.templateId,
          targetId: b.targetId,
        })),
      };
      if (editingTask) {
        await updateSubscriptionTask(editingTask.id, body);
        message.success("已更新");
      } else {
        await createSubscriptionTask(body);
        message.success("已创建");
      }
      setDrawerOpen(false);
      fetchTasks();
    } catch (err: any) {
      if (err?.errorFields) return;
      message.error(err?.message ?? "操作失败");
    }
  };

  const handleDelete = (task: SubscriptionTask) => {
    modal.confirm({
      title: `删除订阅任务「${task.name}」？`,
      content: "删除后不可恢复。GCP 上的 topic / subscription 不会被删除。",
      okText: "删除",
      okType: "danger",
      onOk: async () => {
        await deleteSubscriptionTask(task.id);
        message.success("已删除");
        fetchTasks();
      },
    });
  };

  const handleToggle = async (task: SubscriptionTask) => {
    try {
      if (task.enabled) {
        await pauseSubscriptionTask(task.id);
        message.success("已暂停");
      } else {
        await resumeSubscriptionTask(task.id);
        message.success("已启用");
      }
      fetchTasks();
    } catch {
      message.error("操作失败");
    }
  };

  const describeBinding = useCallback(
    (
      templateId: string,
      version: number | null | undefined,
      targetId: string,
    ) => {
      const tpl = templateMap.get(templateId);
      const tplName = tpl?.name ?? templateId.slice(0, 8);
      const tplLabel = version ? `${tplName} v${version}` : tplName;
      const tgtName = targetMap.get(targetId)?.name ?? targetId.slice(0, 8);
      return `${tplLabel} → ${tgtName}`;
    },
    [templateMap, targetMap],
  );

  const columns: ColumnsType<SubscriptionTask> = [
    {
      title: "名称",
      dataIndex: "name",
      width: 160,
      ellipsis: true,
    },
    {
      title: "状态",
      dataIndex: "enabled",
      width: 70,
      render: (enabled: boolean) =>
        enabled ? (
          <Tag color="green">启用</Tag>
        ) : (
          <Tag color="default">暂停</Tag>
        ),
    },
    {
      title: "流水线",
      dataIndex: "pipelineBindings",
      width: 150,
      render: (bindings: SubscriptionTask["pipelineBindings"]) => {
        const list = bindings ?? [];
        if (list.length === 0) return <Text type="secondary">-</Text>;
        const lines = list.map((b) =>
          describeBinding(b.templateId, b.templateVersion, b.targetId),
        );
        return (
          <Tooltip
            title={lines.join("\n")}
            styles={{ root: { whiteSpace: "pre-line" } }}
          >
            <Text>{list.length} 个模板</Text>
          </Tooltip>
        );
      },
    },
    {
      title: "订阅",
      dataIndex: "subscriptionId",
      width: 170,
      ellipsis: true,
      render: (sub: string, r) => (
        <Tooltip title={`${r.projectId} / ${sub}`}>
          <Text style={{ fontFamily: "monospace", fontSize: 12 }}>{sub}</Text>
        </Tooltip>
      ),
    },
    {
      title: "间隔",
      dataIndex: "pullIntervalSeconds",
      width: 60,
      render: (v: number) => `${v}s`,
    },
    {
      title: "最近状态",
      dataIndex: "lastRunStatus",
      width: 80,
      render: (status: string) => {
        if (!status) return <Text type="secondary">-</Text>;
        const s = STATUS_MAP[status] ?? { color: "default", label: status };
        return <Tag color={s.color}>{s.label}</Tag>;
      },
    },
    {
      title: "最近批次",
      dataIndex: "lastBatchIds",
      width: 130,
      render: (ids: string[] | undefined) => {
        const list = ids ?? [];
        if (list.length === 0) return <Text type="secondary">-</Text>;
        return (
          <Space size={4} wrap>
            {list.slice(0, 2).map((id) => (
              <Link key={id} to={`/pipeline/batch/${id}`}>
                <Text style={{ fontFamily: "monospace", fontSize: 12 }}>
                  {id.slice(0, 8)}
                </Text>
              </Link>
            ))}
            {list.length > 2 && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                +{list.length - 2}
              </Text>
            )}
          </Space>
        );
      },
    },
    {
      title: "最近成功",
      dataIndex: "lastSuccessAt",
      width: 140,
      render: (v: string) =>
        v ? (
          <Text style={{ fontFamily: "monospace", fontSize: 12 }}>
            {new Date(v).toLocaleString("zh-CN")}
          </Text>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: "操作",
      width: 100,
      fixed: "right",
      render: (_: unknown, task: SubscriptionTask) => (
        <Space size={4}>
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => openEdit(task)}
            />
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                {
                  key: "toggle",
                  icon: task.enabled ? (
                    <PauseCircleOutlined />
                  ) : (
                    <PlayCircleOutlined />
                  ),
                  label: task.enabled ? "暂停" : "启用",
                  onClick: () => handleToggle(task),
                },
                {
                  key: "delete",
                  icon: <DeleteOutlined />,
                  label: "删除",
                  danger: true,
                  onClick: () => handleDelete(task),
                },
              ],
            }}
            trigger={["click"]}
          >
            <Button type="text" size="small" icon={<MoreOutlined />} />
          </Dropdown>
        </Space>
      ),
    },
  ];

  return (
    <Card
      title="订阅任务"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchTasks}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建
          </Button>
        </Space>
      }
      styles={{ body: { padding: 0 } }}
    >
      <Table<SubscriptionTask>
        rowKey="id"
        columns={columns}
        dataSource={tasks}
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: false }}
        scroll={{ x: 1160 }}
        size="small"
      />

      <Drawer
        title={editingTask ? "编辑订阅任务" : "新建订阅任务"}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={560}
        extra={
          <Button type="primary" onClick={handleSubmit}>
            {editingTask ? "保存" : "创建"}
          </Button>
        }
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: "请输入任务名称" }]}
          >
            <Input placeholder="e.g. youxin-ingest" />
          </Form.Item>

          <Form.Item
            name="projectId"
            label="GCP 项目 ID"
            rules={[{ required: true, message: "请输入 GCP 项目 ID" }]}
          >
            <Input placeholder="e.g. green-valley-442103" />
          </Form.Item>

          <Form.Item
            name="subscriptionId"
            label="Pub/Sub 订阅 ID"
            tooltip="在 GCP 上创建好 topic + subscription 后，把 subscription ID 填在这里"
            rules={[{ required: true, message: "请输入订阅 ID" }]}
          >
            <Input placeholder="e.g. databrew-ingest-youxin-sub" />
          </Form.Item>

          <Space size={16} style={{ display: "flex" }}>
            <Form.Item
              name="pullIntervalSeconds"
              label="拉取间隔（秒）"
              style={{ flex: 1 }}
            >
              <InputNumber min={1} max={3600} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item
              name="maxMessagesPerPull"
              label="单次最大消息数"
              style={{ flex: 1 }}
            >
              <InputNumber min={1} max={10000} style={{ width: "100%" }} />
            </Form.Item>
          </Space>

          <Divider orientation="left" plain>
            流水线绑定
          </Divider>
          <Text type="secondary" style={{ display: "block", marginBottom: 12 }}>
            每条消息会对下面的<b>每个模板各下发一个批次</b>。
          </Text>

          <Form.List name="pipelineBindings">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...rest }) => (
                  <Space
                    key={key}
                    align="baseline"
                    style={{ display: "flex", marginBottom: 8 }}
                  >
                    <Form.Item
                      {...rest}
                      name={[name, "templateId"]}
                      rules={[{ required: true, message: "选择模板" }]}
                      style={{ marginBottom: 0, minWidth: 220 }}
                    >
                      <Select
                        showSearch
                        placeholder="模板"
                        optionFilterProp="label"
                        options={templateOptions}
                      />
                    </Form.Item>
                    <Form.Item
                      {...rest}
                      name={[name, "templateVersion"]}
                      style={{ marginBottom: 0 }}
                    >
                      <InputNumber
                        min={1}
                        placeholder="最新"
                        style={{ width: 80 }}
                      />
                    </Form.Item>
                    <Form.Item
                      {...rest}
                      name={[name, "targetId"]}
                      rules={[{ required: true, message: "选择资源池" }]}
                      style={{ marginBottom: 0, minWidth: 150 }}
                    >
                      <Select
                        showSearch
                        placeholder="资源池"
                        optionFilterProp="label"
                        options={targetOptions}
                      />
                    </Form.Item>
                    {fields.length > 1 && (
                      <MinusCircleOutlined onClick={() => remove(name)} />
                    )}
                  </Space>
                ))}
                <Button
                  type="dashed"
                  onClick={() => add({ templateId: "", targetId: "" })}
                  block
                  icon={<PlusOutlined />}
                >
                  添加模板
                </Button>
              </>
            )}
          </Form.List>
        </Form>
      </Drawer>
    </Card>
  );
}
