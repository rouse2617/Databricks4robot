import { Typography } from "antd";
import { Link } from "react-router-dom";
import type { RegisteredComponent } from "./types";

const { Text } = Typography;

interface Props {
	components: RegisteredComponent[];
	onDragStart: (e: React.DragEvent, comp: RegisteredComponent) => void;
}

export function ComponentPalette({ components, onDragStart }: Props) {
	return (
		<aside className="palette">
			<div className="palette-header">
				<h3>组件</h3>
				<span className="palette-count">{components.length}</span>
			</div>
			{components.map((c) => (
				<button
					key={c.id}
					type="button"
					className="palette-item"
					draggable
					onDragStart={(e) => onDragStart(e, c)}
				>
					<div className="pi-content">
						<div className="pi-label">{c.name}</div>
						<div className="pi-image">{c.image}</div>
					</div>
				</button>
			))}
			{components.length === 0 && (
				<Text type="secondary" style={{ padding: "24px 8px", display: "block" }}>
					暂无组件，请先在{" "}
					<Link to="/components">步骤组件</Link> 中创建。
				</Text>
			)}
		</aside>
	);
}
