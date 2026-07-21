import {
  DeleteOutlined,
  EditOutlined,
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

interface TaskFormValues {
  name: string;
  templateId: string;
  templateVersion?: number;
  targetId: string;
  projectId: string;
  subscriptionId: string;
  pullIntervalSeconds?: number;
  maxMessagesPerPull?: number;
}

const STATUS_MAP: Record<string, { color: string; label: string }> = {
  succeeded: { color: "green", label: "成功" },
  failed: { color: "red", label: "失败" },
  empty: { color: "default", label: "无消息" },
};

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

  const openCreate = () => {
    setEditingTask(null);
    form.resetFields();
    form.setFieldsValue({
      pullIntervalSeconds: 10,
      maxMessagesPerPull: 1000,
      projectId: "co-prod-gv-cybercap",
    });
    setDrawerOpen(true);
  };

  const openEdit = (task: SubscriptionTask) => {
    setEditingTask(task);
    form.setFieldsValue({
      name: task.name,
      templateId: task.templateId,
      templateVersion: task.templateVersion ?? undefined,
      targetId: task.targetId,
      projectId: task.projectId,
      subscriptionId: task.subscriptionId,
      pullIntervalSeconds: task.pullIntervalSeconds,
      maxMessagesPerPull: task.maxMessagesPerPull,
    });
    setDrawerOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const vals = await form.validateFields();
      const body: SubscriptionTaskCreateRequest = {
        name: vals.name,
        templateId: vals.templateId,
        templateVersion: vals.templateVersion,
        targetId: vals.targetId,
        projectId: vals.projectId,
        subscriptionId: vals.subscriptionId,
        pullIntervalSeconds: vals.pullIntervalSeconds ?? 10,
        maxMessagesPerPull: vals.maxMessagesPerPull ?? 1000,
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
      content: "删除后不可恢复",
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

  const columns: ColumnsType<SubscriptionTask> = [
    {
      title: "名称",
      dataIndex: "name",
      width: 180,
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
      title: "模板",
      dataIndex: "templateId",
      width: 160,
      ellipsis: true,
      render: (id: string, r) => {
        const tpl = templateMap.get(id);
        const name = tpl?.name ?? id.slice(0, 12);
        return r.templateVersion ? `${name} v${r.templateVersion}` : name;
      },
    },
    {
      title: "资源池",
      dataIndex: "targetId",
      width: 140,
      ellipsis: true,
      render: (id: string) => targetMap.get(id)?.name ?? id.slice(0, 12),
    },
    {
      title: "订阅",
      dataIndex: "subscriptionId",
      width: 180,
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
      dataIndex: "lastBatchId",
      width: 100,
      ellipsis: true,
      render: (id: string) =>
        id ? (
          <Link to={`/pipeline/batch/${id}`}>
            <Text style={{ fontFamily: "monospace", fontSize: 12 }}>
              {id.slice(0, 8)}
            </Text>
          </Link>
        ) : (
          <Text type="secondary">-</Text>
        ),
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
        scroll={{ x: 1200 }}
        size="small"
      />

      <Drawer
        title={editingTask ? "编辑订阅任务" : "新建订阅任务"}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={520}
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
            name="templateId"
            label="模板"
            rules={[{ required: true, message: "请选择模板" }]}
          >
            <Select
              showSearch
              placeholder="选择流水线模板"
              optionFilterProp="label"
              options={templates.map((t) => ({
                value: t.id,
                label: `${t.name} (v${t.version ?? "?"})`,
              }))}
            />
          </Form.Item>

          <Form.Item name="templateVersion" label="模板版本">
            <InputNumber
              min={1}
              placeholder="留空使用最新版本"
              style={{ width: "100%" }}
            />
          </Form.Item>

          <Form.Item
            name="targetId"
            label="资源池"
            rules={[{ required: true, message: "请选择资源池" }]}
          >
            <Select
              showSearch
              placeholder="选择执行目标"
              optionFilterProp="label"
              options={targets.map((t) => ({
                value: t.id,
                label: t.name,
              }))}
            />
          </Form.Item>

          <Form.Item
            name="projectId"
            label="GCP 项目 ID"
            rules={[{ required: true, message: "请输入 GCP 项目 ID" }]}
          >
            <Input placeholder="e.g. co-prod-gv-cybercap" />
          </Form.Item>

          <Form.Item
            name="subscriptionId"
            label="Pub/Sub 订阅 ID"
            rules={[{ required: true, message: "请输入订阅 ID" }]}
          >
            <Input placeholder="e.g. databrew-ingest-youxin-sub" />
          </Form.Item>

          <Form.Item name="pullIntervalSeconds" label="拉取间隔（秒）">
            <InputNumber min={1} max={3600} style={{ width: "100%" }} />
          </Form.Item>

          <Form.Item name="maxMessagesPerPull" label="单次最大消息数">
            <InputNumber min={1} max={10000} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Drawer>
    </Card>
  );
}
