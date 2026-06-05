import { Drawer, Spin } from "antd";
import type { McapFile } from "../../api/types";
import McapDetailContent, { McapDetailActions } from "./McapDetailContent";

interface McapDetailDrawerProps {
	open: boolean;
	mcapFile: McapFile | null;
	onClose: () => void;
}

export default function McapDetailDrawer({
	open,
	mcapFile,
	onClose,
}: McapDetailDrawerProps) {
	return (
		<Drawer
			title="MCAP 文件详情"
			placement="right"
			width={720}
			open={open}
			onClose={onClose}
			extra={mcapFile ? <McapDetailActions mcapFile={mcapFile} /> : null}
		>
			{mcapFile ? (
				<McapDetailContent mcapFile={mcapFile} onNavigateAway={onClose} />
			) : (
				<Spin />
			)}
		</Drawer>
	);
}
