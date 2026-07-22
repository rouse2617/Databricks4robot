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
  Descriptions,
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
  type DispatchBatch,
  type DispatchBatchItem,
  type DispatchRun,
  listDispatchBatchItems,
  listSubscriptionTaskBatches,
  listSubscriptionTaskRuns,
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

// Backfill job / item status → tag color, for the dispatch-history drawer.
const BATCH_STATUS_COLOR: Record<string, string> = {
  completed: "green",
  running: "blue",
  pending: "default",
  failed: "red",
  paused: "orange",
  cancelled: "default",
};
const ITEM_STATUS_COLOR: Record<string, string> = {
  completed: "green",
  running: "blue",
  submitted: "blue",
  pending: "default",
  failed: "red",
  cancelled: "default",
};

// A unified dispatch-history row: either a batch (≥2 assets, expandable to its
// per-asset items) or a single run (1 asset).
interface HistoryRow {
  key: string;
  kind: "run" | "batch";
  id: string;
  name: string;
  status: string;
  assetSummary: string;
  createdAt: string;
  to: string;
}

export function SubscriptionTasksPanel() {
  const { modal } = App.useApp();
  const [tasks, setTasks] = useState<SubscriptionTask[]>([]);
  const [loading, setLoading] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<SubscriptionTask | null>(null);
  const [form] = Form.useForm<TaskFormValues>();

  // Dispatch-history drawer: batches (CYB-3798) + single runs (CYB-3801) a task
  // dispatched, merged; batches expand to their per-asset items.
  const [historyTask, setHistoryTask] = useState<SubscriptionTask | null>(null);
  const [batches, setBatches] = useState<DispatchBatch[]>([]);
  const [runs, setRuns] = useState<DispatchRun[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [itemsByBatch, setItemsByBatch] = useState<
    Record<string, DispatchBatchItem[]>
  >({});
  const [itemsLoading, setItemsLoading] = useState<Record<string, boolean>>({});

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

  // Merge batches + single runs into one newest-first history list.
  const historyRows = useMemo<HistoryRow[]>(() => {
    const rows: HistoryRow[] = [
      ...batches.map((b) => ({
        key: `b:${b.id}`,
        kind: "batch" as const,
        id: b.id,
        name: b.name || b.id.slice(0, 12),
        status: b.status,
        assetSummary: `${b.completedCount}/${b.totalCount} 成功${
          b.failedCount ? `，${b.failedCount} 失败` : ""
        }`,
        createdAt: b.createdAt,
        to: `/pipeline/batch/${b.id}`,
      })),
      ...runs.map((r) => ({
        key: `r:${r.id}`,
        kind: "run" as const,
        id: r.id,
        name: r.pipelineName || r.id.slice(0, 12),
        status: r.status,
        assetSummary: r.assetIds?.[0] ?? "1 资产",
        createdAt: r.createdAt,
        to: `/runs/${r.id}`,
      })),
    ];
    rows.sort((a, b) =>
      a.createdAt < b.createdAt ? 1 : a.createdAt > b.createdAt ? -1 : 0,
    );
    return rows;
  }, [batches, runs]);

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

  const openHistory = useCallback(async (task: SubscriptionTask) => {
    setHistoryTask(task);
    setBatches([]);
    setRuns([]);
    setItemsByBatch({});
    setHistoryLoading(true);
    try {
      const [b, r] = await Promise.all([
        listSubscriptionTaskBatches(task.id),
        listSubscriptionTaskRuns(task.id),
      ]);
      setBatches(b);
      setRuns(r);
    } catch {
      message.error("加载下发历史失败");
    } finally {
      setHistoryLoading(false);
    }
  }, []);

  const loadBatchItems = useCallback(async (batchId: string) => {
    setItemsLoading((m) => ({ ...m, [batchId]: true }));
    try {
      const items = await listDispatchBatchItems(batchId);
      setItemsByBatch((m) => ({ ...m, [batchId]: items }));
    } catch {
      message.error("加载资产明细失败");
    } finally {
      setItemsLoading((m) => ({ ...m, [batchId]: false }));
    }
  }, []);

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
      render: (name: string, task) => (
        <Tooltip title="查看下发历史与资产">
          <Button
            type="link"
            size="small"
            style={{ padding: 0, height: "auto" }}
            onClick={() => openHistory(task)}
          >
            {name}
          </Button>
        </Tooltip>
      ),
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
            每条消息会对下面的<b>每个模板各下发一次</b>（1 个资产→单 run，多个→批次）。
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

      <Drawer
        title={historyTask ? `下发历史 · ${historyTask.name}` : "下发历史"}
        open={historyTask !== null}
        onClose={() => setHistoryTask(null)}
        width={760}
      >
        {historyTask && (
          <>
            <Descriptions size="small" column={1} bordered style={{ marginBottom: 16 }}>
              <Descriptions.Item label="订阅">
                <Text style={{ fontFamily: "monospace", fontSize: 12 }}>
                  {historyTask.subscriptionId}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item label="流水线绑定">
                {(historyTask.pipelineBindings ?? []).length} 个模板（1 资产→run，多资产→批次）
              </Descriptions.Item>
              <Descriptions.Item label="最近状态">
                {historyTask.lastRunStatus
                  ? STATUS_MAP[historyTask.lastRunStatus]?.label ??
                    historyTask.lastRunStatus
                  : "-"}
              </Descriptions.Item>
            </Descriptions>
            <Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
              该订阅任务下发过的批次（≥2 资产）和单个 run（1 资产）。批次可展开看每批跑过的资产。
            </Text>
            <Table<HistoryRow>
              rowKey="key"
              size="small"
              loading={historyLoading}
              dataSource={historyRows}
              pagination={{ pageSize: 10, hideOnSinglePage: true }}
              locale={{ emptyText: "还没有下发过" }}
              columns={[
                {
                  title: "类型",
                  dataIndex: "kind",
                  width: 64,
                  render: (k: HistoryRow["kind"]) =>
                    k === "batch" ? (
                      <Tag color="blue">批次</Tag>
                    ) : (
                      <Tag color="geekblue">单 run</Tag>
                    ),
                },
                {
                  title: "名称",
                  dataIndex: "name",
                  ellipsis: true,
                  render: (name: string, row) => (
                    <Link to={row.to}>
                      <Text style={{ fontSize: 12 }}>{name}</Text>
                    </Link>
                  ),
                },
                {
                  title: "状态",
                  dataIndex: "status",
                  width: 90,
                  render: (s: string) => (
                    <Tag color={BATCH_STATUS_COLOR[s] ?? ITEM_STATUS_COLOR[s] ?? "default"}>
                      {s}
                    </Tag>
                  ),
                },
                {
                  title: "资产",
                  dataIndex: "assetSummary",
                  ellipsis: true,
                  render: (v: string) => (
                    <Text style={{ fontSize: 12, fontFamily: "monospace" }}>{v}</Text>
                  ),
                },
                {
                  title: "时间",
                  dataIndex: "createdAt",
                  width: 160,
                  render: (v: string) => (
                    <Text style={{ fontFamily: "monospace", fontSize: 12 }}>
                      {new Date(v).toLocaleString("zh-CN")}
                    </Text>
                  ),
                },
              ]}
              expandable={{
                rowExpandable: (row) => row.kind === "batch",
                onExpand: (expanded, row) => {
                  if (expanded && row.kind === "batch" && !itemsByBatch[row.id])
                    loadBatchItems(row.id);
                },
                expandedRowRender: (row) =>
                  row.kind === "batch" ? (
                    <Table<DispatchBatchItem>
                      rowKey="id"
                      size="small"
                      loading={itemsLoading[row.id]}
                      dataSource={itemsByBatch[row.id] ?? []}
                      pagination={{ pageSize: 20, hideOnSinglePage: true }}
                      locale={{ emptyText: "无资产" }}
                      columns={[
                        {
                          title: "资产 ID",
                          dataIndex: "assetId",
                          ellipsis: true,
                          render: (a: string) => (
                            <Text style={{ fontFamily: "monospace", fontSize: 12 }}>
                              {a}
                            </Text>
                          ),
                        },
                        {
                          title: "状态",
                          dataIndex: "status",
                          width: 90,
                          render: (s: string) => (
                            <Tag color={ITEM_STATUS_COLOR[s] ?? "default"}>{s}</Tag>
                          ),
                        },
                        {
                          title: "错误",
                          dataIndex: "errorMessage",
                          ellipsis: true,
                          render: (e?: string | null) =>
                            e ? (
                              <Tooltip title={e}>
                                <Text type="danger" style={{ fontSize: 12 }}>
                                  {e}
                                </Text>
                              </Tooltip>
                            ) : (
                              <Text type="secondary">-</Text>
                            ),
                        },
                      ]}
                    />
                  ) : null,
              }}
            />
          </>
        )}
      </Drawer>
    </Card>
  );
}
