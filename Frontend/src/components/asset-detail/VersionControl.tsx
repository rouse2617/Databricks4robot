import { BranchesOutlined, CheckCircleFilled, DownOutlined, HistoryOutlined } from "@ant-design/icons";
import { Dropdown, Skeleton, Tag, Tooltip, Typography } from "antd";
import type { MenuProps } from "antd";
import type { RevisionSummary } from "../../api/types";
import { formatShortDateTime } from "../../lib/dateTime";

const { Text } = Typography;

export interface VersionControlProps {
	assetId: string;
	revisions: RevisionSummary[];
	onSelect: (nextAssetId: string) => void;
	loading?: boolean;
}

function revisionLabel(revision: RevisionSummary): string {
	const current = revision.is_current ? " (当前)" : "";
	return `v${revision.revision}${current}`;
}

export default function VersionControl({
	assetId,
	revisions,
	onSelect,
	loading = false,
}: VersionControlProps) {
	if (loading) {
		return <Skeleton.Input active size="small" style={{ width: 180, height: 32 }} />;
	}

	const sorted = [...revisions].sort((a, b) => b.revision - a.revision);
	const current = sorted.find((r) => r.is_current) ?? sorted[0];

	if (!current || sorted.length <= 1) {
		const rev = current?.revision ?? 1;
		return (
			<span
				className="inline-flex items-center gap-1.5 h-7 px-2 rounded-md bg-[#F1F5FD] text-[#64748B] text-xs"
				aria-label="资产版本"
			>
				<BranchesOutlined className="text-[11px]" />
				<span>v{rev}</span>
				<Tag color="default" className="m-0 text-[10px] leading-[16px] px-1 border-0 bg-transparent">
					仅一版
				</Tag>
			</span>
		);
	}

	const menuItems: MenuProps["items"] = sorted.map((r, idx) => ({
		key: r.asset_id,
		label: (
			<div className="py-1" style={{ minWidth: 280 }}>
				<div className="flex items-start gap-3">
					{/* Timeline indicator */}
					<div className="flex flex-col items-center pt-0.5">
						<div
							className="w-2.5 h-2.5 rounded-full border-2 shrink-0"
							style={{
								borderColor: r.asset_id === assetId ? "#2563EB" : r.is_current ? "#059669" : "#CBD5E1",
								background: r.asset_id === assetId ? "#2563EB" : r.is_current ? "#059669" : "#fff",
							}}
						/>
						{idx < sorted.length - 1 && (
							<div className="w-px h-6 bg-[#E2E8F0] mt-1" />
						)}
					</div>
					{/* Content */}
					<div className="min-w-0 flex-1">
						<div className="flex items-center gap-2">
							<Text strong={r.asset_id === assetId} className="text-sm">
								v{r.revision}
							</Text>
							{r.is_current ? (
								<Tag
									className="m-0 text-[10px] leading-[16px] px-1.5"
									style={{ color: "#059669", background: "#ECFDF5", borderColor: "#A7F3D0" }}
								>
									<CheckCircleFilled className="mr-0.5" />
									当前
								</Tag>
							) : null}
							{r.asset_id === assetId && !r.is_current ? (
								<Tag color="default" className="m-0 text-[10px] leading-[16px] px-1.5">
									<HistoryOutlined className="mr-0.5" />
									查看中
								</Tag>
							) : null}
						</div>
						<div className="flex items-center gap-1.5 mt-0.5">
							<Text
								type="secondary"
								className="text-xs font-mono"
								copyable={{ text: r.asset_id }}
							>
								{r.asset_id}
							</Text>
							<span className="text-[#CBD5E1]">·</span>
							<Text type="secondary" className="text-xs">
								{formatShortDateTime(r.created_at)}
							</Text>
						</div>
					</div>
				</div>
			</div>
		),
	}));

	const active = sorted.find((r) => r.asset_id === assetId) ?? current;

	return (
		<Dropdown
			menu={{
				items: menuItems,
				selectable: true,
				selectedKeys: [assetId],
				onClick: ({ key }) => {
					if (key && key !== assetId) {
						onSelect(String(key));
					}
				},
			}}
			trigger={["click"]}
			overlayStyle={{ minWidth: 300 }}
			overlayClassName="version-control-dropdown"
		>
			<button
				type="button"
				className="inline-flex items-center gap-1.5 h-8 px-3 rounded-lg border border-[#E2E8F0] bg-white text-sm text-[#1E293B] hover:border-[#2563EB] hover:text-[#2563EB] hover:shadow-sm transition-all cursor-pointer"
				aria-label="资产版本"
				aria-haspopup="listbox"
			>
				<BranchesOutlined className="text-[#64748B] text-xs" />
				<Text type="secondary" className="text-xs">
					版本
				</Text>
				<Text strong className="text-sm">
					{revisionLabel(active)}
				</Text>
				{sorted.length > 2 && (
					<Tooltip title={`共 ${sorted.length} 个版本`}>
						<span className="text-[10px] text-[#94A3B8] bg-[#F1F5FD] px-1 rounded">
							{sorted.length}
						</span>
					</Tooltip>
				)}
				<DownOutlined className="text-[10px] text-[#64748B] ml-0.5" />
			</button>
		</Dropdown>
	);
}
