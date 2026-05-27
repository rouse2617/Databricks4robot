import { Button, Input, Typography } from "antd";
import { useState } from "react";
import type { RegisteredComponent } from "./types";
import { PlusOutlined } from "@ant-design/icons";

const { Text } = Typography;

interface Props {
	components: RegisteredComponent[];
	onChange: (c: RegisteredComponent[]) => void;
}

function blank(): RegisteredComponent {
	return {
		id: "c" + Date.now(),
		name: "",
		image: "",
		command: ["sh", "-c"],
		args: [],
		cpu: "",
		memory: "",
		disk: "",
	};
}

export function ComponentManager({ components, onChange }: Props) {
	const [editing, setEditing] = useState<RegisteredComponent | null>(null);
	const [isNew, setIsNew] = useState(false);

	const save = (c: RegisteredComponent) => {
		if (isNew) {
			onChange([...components, c]);
		} else {
			onChange(components.map((x) => (x.id === c.id ? c : x)));
		}
		setEditing(null);
	};

	const remove = (id: string) => {
		onChange(components.filter((x) => x.id !== id));
		if (editing?.id === id) setEditing(null);
	};

	return (
		<div className="component-manager">
			<div className="cm-header">
				<h3>组件注册表</h3>
				<Button
					type="primary"
					size="small"
					icon={<PlusOutlined />}
					onClick={() => {
						setEditing(blank());
						setIsNew(true);
					}}
				>
					新建
				</Button>
			</div>

			<div className="cm-list">
				{components.length === 0 && (
					<Text type="secondary" style={{ padding: 40, textAlign: "center", display: "block" }}>
						暂无注册组件
					</Text>
				)}
				{components.map((c) => (
					<div
						key={c.id}
						className={`cm-item ${editing?.id === c.id ? "active" : ""}`}
						onClick={() => {
							setEditing(c);
							setIsNew(false);
						}}
					>
						<div className="cm-item-name">{c.name}</div>
						<div className="cm-item-image">{c.image}</div>
					</div>
				))}
			</div>

			{editing && (
				<div className="cm-form">
					<h4>{isNew ? "新建组件" : "编辑组件"}</h4>
					<div className="cm-form-fields">
						<label>
							名称
							<Input
								value={editing.name}
								onChange={(e) => setEditing({ ...editing, name: e.target.value })}
								placeholder="my-component"
							/>
						</label>
						<label>
							镜像
							<Input
								value={editing.image}
								onChange={(e) => setEditing({ ...editing, image: e.target.value })}
								placeholder="repo/image:tag"
							/>
						</label>
						<label>
							命令 (JSON)
							<Input
								value={JSON.stringify(editing.command)}
								onChange={(e) => {
									try {
										setEditing({ ...editing, command: JSON.parse(e.target.value) });
									} catch {
										/* ignore */
									}
								}}
								placeholder='["sh", "-c"]'
							/>
						</label>
						<label>
							CPU
							<Input
								value={editing.cpu}
								onChange={(e) => setEditing({ ...editing, cpu: e.target.value })}
								placeholder="500m"
							/>
						</label>
						<label>
							内存
							<Input
								value={editing.memory}
								onChange={(e) => setEditing({ ...editing, memory: e.target.value })}
								placeholder="256Mi"
							/>
						</label>
						<label>
							磁盘
							<Input
								value={editing.disk}
								onChange={(e) => setEditing({ ...editing, disk: e.target.value })}
								placeholder="1Gi"
							/>
						</label>
					</div>
					<div className="cm-form-actions">
						<Button type="primary" onClick={() => save(editing)}>
							{isNew ? "创建" : "保存"}
						</Button>
						{!isNew && (
							<Button danger onClick={() => remove(editing.id)}>
								删除
							</Button>
						)}
						<Button onClick={() => setEditing(null)}>取消</Button>
					</div>
				</div>
			)}
		</div>
	);
}
