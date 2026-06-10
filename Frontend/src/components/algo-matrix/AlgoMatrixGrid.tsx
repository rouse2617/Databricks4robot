import { Table } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useNavigate } from "react-router-dom";
import type { AlgoRegistryItem } from "../../api/algoRegistry";
import type { Asset } from "../../api/types";
import {
	getAlgoResultDetail,
	getAlgoStatusFromResults,
} from "../../lib/algoStatus";
import { navigateToAssetDetail } from "../../lib/assets/assetWorkbenchNavigation";
import AlgoStatusCell from "./AlgoStatusCell";
import AlgoStatusPopover from "./AlgoStatusPopover";

interface AlgoMatrixGridProps {
	assets: Asset[];
	algorithms: AlgoRegistryItem[];
	loading: boolean;
	total: number;
	page: number;
	pageSize: number;
	onPageChange: (page: number, pageSize: number) => void;
	onRefresh: () => void;
}

export default function AlgoMatrixGrid({
	assets,
	algorithms,
	loading,
	total,
	page,
	pageSize,
	onPageChange,
	onRefresh,
}: AlgoMatrixGridProps) {
	const navigate = useNavigate();
	const columns: ColumnsType<Asset> = [
		{
			title: "Asset ID",
			dataIndex: "asset_id",
			key: "asset_id",
			width: 220,
			fixed: "left",
			ellipsis: true,
			render: (id: string) => (
				<a
					href={`/assets/${id}`}
					style={{ fontFamily: "monospace", fontSize: 12 }}
					onClick={(e) => {
						if (
							e.ctrlKey ||
							e.metaKey ||
							e.shiftKey ||
							e.altKey ||
							e.button !== 0
						)
							return;
						e.preventDefault();
						navigateToAssetDetail(navigate, id);
					}}
				>
					{id.slice(0, 12)}…
				</a>
			),
		},
		...algorithms.map((algo) => ({
			title: (
				<div style={{ textAlign: "center", fontSize: 12 }}>
					<div>{algo.name}</div>
					<div style={{ color: "#64748b", fontSize: 10 }}>{algo.version}</div>
				</div>
			),
			key: algo.key,
			width: 80,
			align: "center" as const,
			render: (_: unknown, asset: Asset) => {
				const detail = getAlgoResultDetail(asset.algo_results, algo.key);
				const status = getAlgoStatusFromResults(asset.algo_results, algo.key);
				return (
					<AlgoStatusPopover
						assetId={asset.asset_id}
						algoKey={algo.key}
						detail={detail}
						onReset={onRefresh}
					>
						<span>
							<AlgoStatusCell status={status} onClick={() => {}} />
						</span>
					</AlgoStatusPopover>
				);
			},
		})),
	];

	return (
		<Table<Asset>
			columns={columns}
			dataSource={assets}
			rowKey="asset_id"
			loading={loading}
			scroll={{ x: 220 + algorithms.length * 80 }}
			size="small"
			pagination={{
				current: page,
				pageSize,
				total,
				showSizeChanger: true,
				showQuickJumper: total > 200,
				pageSizeOptions: ["20", "50", "100"],
				showTotal: (t) => `共 ${t} 条`,
				onChange: onPageChange,
			}}
		/>
	);
}
