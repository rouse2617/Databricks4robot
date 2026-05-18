// ─── SaveViewDialog — Modal to name and save current queryState as a view ───
// Validates: Requirements R11

import { Input, Modal } from "antd";
import { useState } from "react";

export interface SaveViewDialogProps {
	open: boolean;
	onSave: (name: string) => void;
	onCancel: () => void;
}

export default function SaveViewDialog({
	open,
	onSave,
	onCancel,
}: SaveViewDialogProps) {
	const [name, setName] = useState("");

	const handleOk = () => {
		const trimmed = name.trim();
		if (!trimmed) return;
		onSave(trimmed);
		setName("");
	};

	const handleCancel = () => {
		setName("");
		onCancel();
	};

	return (
		<Modal
			title="保存查询"
			open={open}
			onOk={handleOk}
			onCancel={handleCancel}
			okText="保存"
			cancelText="取消"
			okButtonProps={{ disabled: !name.trim() }}
			destroyOnHidden
		>
			<Input
				placeholder="输入查询名称"
				value={name}
				onChange={(e) => setName(e.target.value)}
				onPressEnter={handleOk}
				autoFocus
			/>
		</Modal>
	);
}
