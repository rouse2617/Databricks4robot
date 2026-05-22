import { DownOutlined } from "@ant-design/icons";
import { Dropdown, Skeleton, Tag, Typography } from "antd";
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
		return <Skeleton.Input active size="small" style={{ width: 160, height: 32 }} />;
	}

	const sorted = [...revisions].sort((a, b) => b.revision - a.revision);
	const current = sorted.find((r) => r.is_current) ?? sorted[0];

	if (!current || sorted.length <= 1) {
		const rev = current?.revision ?? 1;
		return (
			<Text type="secondary" className="text-xs" aria-label="资产版本">
				v{rev} · 仅一版
			</Text>
		);
	}

	const menuItems: MenuProps["items"] = sorted.map((r) => ({
		key: r.asset_id,
		label: (
			<div className="py-0.5" style={{ minWidth: 248 }}>
				<div className="flex items-center gap-2">
					<span
						className="inline-block w-1 rounded-sm shrink-0 self-stretch min-h-[20px]"
						style={{
							background: r.asset_id === assetId ? "#2563EB" : "transparent",
						}}
						aria-hidden
					/>
					<div className="min-w-0 flex-1">
						<div className="flex items-center gap-2">
							<Text strong={r.asset_id === assetId}>v{r.revision}</Text>
							{r.is_current ? (
								<Tag color="blue" className="m-0 text-[11px] leading-[18px] px-1">
									当前
								</Tag>
							) : null}
						</div>
						<Text type="secondary" className="text-xs font-mono block truncate">
							{r.asset_id} · {formatShortDateTime(r.created_at)}
						</Text>
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
			overlayStyle={{ minWidth: 280 }}
		>
			<button
				type="button"
				className="inline-flex items-center gap-1.5 h-8 px-2.5 rounded-lg border border-[#E2E8F0] bg-white text-sm text-[#1E293B] hover:border-[#2563EB] hover:text-[#2563EB] transition-colors cursor-pointer"
				aria-label="资产版本"
				aria-haspopup="listbox"
			>
				<Text type="secondary" className="text-xs">
					版本
				</Text>
				<Text strong>{revisionLabel(active)}</Text>
				<DownOutlined className="text-[10px] text-[#64748B]" />
			</button>
		</Dropdown>
	);
}
