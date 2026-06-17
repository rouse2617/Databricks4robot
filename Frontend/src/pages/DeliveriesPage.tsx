import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, message, Select, Space, Table, Tag, Typography } from "antd";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { deliveriesApi } from "../api/deliveries";
import type { Delivery } from "../api/types";
import CreateDeliveryModal from "../components/deliveries/CreateDeliveryModal";
import { formatDateTime } from "../lib/dateTime";
import {
	COLUMN_LABELS,
	formatBusinessStatusLabel,
	resolveBusinessStatusTagColor,
} from "../lib/productVocabulary";

const { Title } = Typography;

const STATUS_OPTIONS = [
	{ label: "全部", value: "" },
	{ label: "草稿", value: "draft" },
	{ label: "待交付", value: "pending" },
	{ label: "已交付", value: "delivered" },
	{ label: "已接受", value: "accepted" },
	{ label: "已拒绝", value: "rejected" },
];

export default function DeliveriesPage() {
	const navigate = useNavigate();
	const [msg, msgCtx] = message.useMessage();
	const [items, setItems] = useState<Delivery[]>([]);
	const [total, setTotal] = useState(0);
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(20);
	const [status, setStatus] = useState("");
	const [loading, setLoading] = useState(false);
	const [modalOpen, setModalOpen] = useState(false);

	// antd v5 useMessage may return a fresh `msg` reference on every render.
	// Capture it in a ref so fetchData's identity only depends on real query
	// params; otherwise useCallback ties to msg → useEffect re-fires every
	// render → infinite API spam.
	const msgRef = useRef(msg);
	msgRef.current = msg;

	const fetchData = useCallback(async () => {
		setLoading(true);
		try {
			const res = await deliveriesApi.list({
				page,
				page_size: pageSize,
				status: status || undefined,
			});
			setItems(res.items ?? []);
			setTotal(res.total);
		} catch {
			msgRef.current.error("加载交付列表失败");
		} finally {
			setLoading(false);
		}
	}, [page, pageSize, status]);

	useEffect(() => {
		fetchData();
	}, [fetchData]);

	const columns = [
		{
			title: "交付 ID",
			dataIndex: "delivery_id",
			key: "delivery_id",
			width: 220,
			ellipsis: true,
			render: (id: string) => (
				<Link
					to={`/deliveries/${id}`}
					onClick={(e) => e.stopPropagation()}
					className="font-mono text-xs"
				>
					{id}
				</Link>
			),
		},
		{
			title: "客户",
			dataIndex: "customer_id",
			key: "customer_id",
			width: 160,
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 100,
			render: (val: string) => (
				<Tag color={resolveBusinessStatusTagColor(val)}>
					{formatBusinessStatusLabel(val)}
				</Tag>
			),
		},
		{
			title: "交付时间",
			dataIndex: "delivered_at",
			key: "delivered_at",
			width: 180,
			render: (val?: string) => formatDateTime(val),
		},
		{
			title: "资产数",
			dataIndex: "asset_count",
			key: "asset_count",
			width: 80,
		},
		{
			title: COLUMN_LABELS.owner,
			dataIndex: "owner",
			key: "owner",
			width: 120,
		},
		{
			title: "创建时间",
			dataIndex: "created_at",
			key: "created_at",
			width: 180,
			render: (val?: string) => formatDateTime(val),
		},
	];

	return (
		<div>
			{msgCtx}
			<Title level={4} style={{ margin: "0 0 16px 0" }}>
				交付管理
			</Title>

			{/* Toolbar: status filter + refresh + create */}
			<Space style={{ marginBottom: 12 }} wrap>
				<Select
					value={status}
					onChange={(v) => {
						setStatus(v);
						setPage(1);
					}}
					options={STATUS_OPTIONS}
					style={{ width: 140 }}
					placeholder="状态过滤"
				/>
				<Button icon={<ReloadOutlined />} onClick={fetchData}>
					刷新
				</Button>
				<Button
					type="primary"
					icon={<PlusOutlined />}
					onClick={() => setModalOpen(true)}
				>
					新建交付
				</Button>
			</Space>

			<Table
				rowKey="delivery_id"
				columns={columns}
				dataSource={items}
				loading={loading}
				scroll={{ x: 1100 }}
				pagination={{
					current: page,
					pageSize,
					total,
					showSizeChanger: true,
					showQuickJumper: total > 200,
					showTotal: (t) => `共 ${t} 条`,
					onChange: (p, ps) => {
						setPage(p);
						setPageSize(ps);
					},
				}}
				onRow={(record) => ({
					onClick: (e) => {
						const el = e.target as HTMLElement;
						if (
							el.closest(
								"a, button, input, textarea, select, label, [role='checkbox'], .ant-pagination, .ant-select, .ant-pagination-item",
							)
						) {
							return;
						}
						navigate(`/deliveries/${record.delivery_id}`);
					},
					style: { cursor: "pointer" },
				})}
				size="middle"
			/>

			<CreateDeliveryModal
				open={modalOpen}
				assetIds={[]}
				onClose={() => setModalOpen(false)}
				onSuccess={(deliveryId) => {
					setModalOpen(false);
					navigate(`/deliveries/${deliveryId}`);
				}}
			/>
		</div>
	);
}
