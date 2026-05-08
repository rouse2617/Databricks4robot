import {
	Button,
	Input,
	message,
	Modal,
	Popconfirm,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
	queryApi,
	type QueryRequest,
	type SavedQuery,
	type QueryValidateResponse,
} from "../api/query";

const { Title, Text } = Typography;

type EditorState = {
	open: boolean;
	mode: "create" | "edit";
	id?: string;
	name: string;
	description: string;
	queryText: string;
	validation: QueryValidateResponse | null;
	validationError: string | null;
	saving: boolean;
};

function defaultQueryRequest(): QueryRequest {
	return {
		schema_version: "v1",
		scope: { resource: "assets" },
		page: { page: 1, page_size: 20 },
		sort: [{ field: "updated_at", direction: "desc" }],
	};
}

function savedQueryToEditorState(item?: SavedQuery): EditorState {
	return {
		open: true,
		mode: item ? "edit" : "create",
		id: item?.saved_query_id,
		name: item?.name ?? "",
		description: item?.description ?? "",
		queryText: JSON.stringify(item?.query_ir_json ?? defaultQueryRequest(), null, 2),
		validation: null,
		validationError: null,
		saving: false,
	};
}

export default function SavedQueriesPage() {
	const navigate = useNavigate();
	const [msg, msgCtx] = message.useMessage();
	const [items, setItems] = useState<SavedQuery[]>([]);
	const [loading, setLoading] = useState(true);
	const [editor, setEditor] = useState<EditorState>({
		open: false,
		mode: "create",
		name: "",
		description: "",
		queryText: "",
		validation: null,
		validationError: null,
		saving: false,
	});

	const load = async () => {
		setLoading(true);
		try {
			const data = await queryApi.listSavedQueries();
			setItems(data.items ?? []);
		} catch (err) {
			void msg.error(err instanceof Error ? err.message : "加载 saved queries 失败");
		} finally {
			setLoading(false);
		}
	};

	useEffect(() => {
		void load();
	}, []);

	const columns = useMemo<ColumnsType<SavedQuery>>(
		() => [
			{
				title: "名称",
				dataIndex: "name",
				render: (value: string) => <Text strong>{value}</Text>,
			},
			{
				title: "描述",
				dataIndex: "description",
				render: (value?: string) => value || "—",
			},
			{
				title: "资源",
				dataIndex: "resource",
				render: (value: string) => <Tag>{value}</Tag>,
			},
			{
				title: "Schema",
				dataIndex: "schema_version",
				render: (value: string) => <Tag color="blue">{value}</Tag>,
			},
			{
				title: "操作",
				key: "actions",
				render: (_value, row) => (
					<Space>
						<Button size="small" onClick={() => setEditor(savedQueryToEditorState(row))}>
							编辑
						</Button>
						<Button
							size="small"
							onClick={() => {
								sessionStorage.setItem("assets_open_saved_query_id", `remote:${row.saved_query_id}`);
								navigate("/assets");
							}}
						>
							在工作台打开
						</Button>
						<Popconfirm
							title="删除 saved query?"
							onConfirm={async () => {
								await queryApi.deleteSavedQuery(row.saved_query_id);
								void msg.success("已删除");
								await load();
							}}
						>
							<Button size="small" danger>
								删除
							</Button>
						</Popconfirm>
					</Space>
				),
			},
		],
		[load, msg, navigate],
	);

	const parseEditorQuery = (): QueryRequest | null => {
		try {
			return JSON.parse(editor.queryText) as QueryRequest;
		} catch {
			return null;
		}
	};

	const validateEditorQuery = async () => {
		const query = parseEditorQuery();
		if (!query) {
			setEditor((prev) => ({
				...prev,
				validation: null,
				validationError: "Query JSON 不是合法 JSON",
			}));
			return;
		}
		try {
			const validation = await queryApi.validate(query);
			setEditor((prev) => ({
				...prev,
				validation,
				validationError: null,
			}));
		} catch (err) {
			setEditor((prev) => ({
				...prev,
				validation: null,
				validationError: err instanceof Error ? err.message : "validate 失败",
			}));
		}
	};

	const saveEditor = async () => {
		const query = parseEditorQuery();
		if (!query) {
			void msg.error("Query JSON 不是合法 JSON");
			return;
		}
		setEditor((prev) => ({ ...prev, saving: true }));
		try {
			await queryApi.validate(query);
			if (editor.mode === "create") {
				await queryApi.createSavedQuery({
					name: editor.name.trim(),
					description: editor.description.trim() || undefined,
					query_ir_json: query,
				});
			} else if (editor.id) {
				await queryApi.updateSavedQuery(editor.id, {
					name: editor.name.trim(),
					description: editor.description.trim() || undefined,
					query_ir_json: query,
				});
			}
			void msg.success(editor.mode === "create" ? "已创建" : "已更新");
			setEditor((prev) => ({ ...prev, open: false, saving: false }));
			await load();
		} catch (err) {
			void msg.error(err instanceof Error ? err.message : "保存失败");
			setEditor((prev) => ({ ...prev, saving: false }));
		}
	};

	return (
		<div>
			{msgCtx}
			<div
				style={{
					display: "flex",
					justifyContent: "space-between",
					alignItems: "center",
					marginBottom: 16,
				}}
			>
				<div>
					<Title level={4} style={{ margin: 0 }}>
						Saved Queries
					</Title>
					<Text type="secondary">集中管理持久化的 Query IR 查询对象。</Text>
				</div>
				<Button type="primary" onClick={() => setEditor(savedQueryToEditorState())}>
					新建 Query
				</Button>
			</div>

			<Table
				rowKey="saved_query_id"
				dataSource={items}
				columns={columns}
				loading={loading}
				pagination={false}
			/>

			<Modal
				open={editor.open}
				title={editor.mode === "create" ? "新建 Saved Query" : "编辑 Saved Query"}
				onCancel={() => setEditor((prev) => ({ ...prev, open: false }))}
				onOk={() => void saveEditor()}
				okButtonProps={{ disabled: !editor.name.trim(), loading: editor.saving }}
				width={860}
			>
				<div style={{ display: "grid", gap: 12 }}>
					<Input
						placeholder="名称"
						value={editor.name}
						onChange={(e) => setEditor((prev) => ({ ...prev, name: e.target.value }))}
					/>
					<Input
						placeholder="描述"
						value={editor.description}
						onChange={(e) =>
							setEditor((prev) => ({ ...prev, description: e.target.value }))
						}
					/>
					<Input.TextArea
						rows={16}
						value={editor.queryText}
						onChange={(e) =>
							setEditor((prev) => ({ ...prev, queryText: e.target.value }))
						}
					/>
					<Space>
						<Button onClick={() => void validateEditorQuery()}>Validate</Button>
					</Space>
					{editor.validationError && (
						<Text type="danger">{editor.validationError}</Text>
					)}
					{editor.validation && (
						<div style={{ display: "grid", gap: 8 }}>
							<Text strong>Warnings</Text>
							<Text type="secondary">
								{editor.validation.warnings?.length
									? editor.validation.warnings.join("；")
									: "无"}
							</Text>
							<Text strong>Debug Plan</Text>
							<pre
								style={{
									margin: 0,
									padding: 12,
									background: "#0F172A",
									color: "#E2E8F0",
									borderRadius: 8,
									overflowX: "auto",
									fontSize: 12,
								}}
							>
								{JSON.stringify(editor.validation.debug_plan ?? {}, null, 2)}
							</pre>
						</div>
					)}
				</div>
			</Modal>
		</div>
	);
}
