import { Alert, Button, Spin } from "antd";
import { Link } from "react-router-dom";
import { PipelineEmptyState } from "./PipelineEmptyState";
import type { RegisteredComponent } from "./types";

interface Props {
	components: RegisteredComponent[];
	onDragStart: (e: React.DragEvent, comp: RegisteredComponent) => void;
	onAddComponent?: (comp: RegisteredComponent) => void;
	loading?: boolean;
	error?: string | null;
	onRetry?: () => void;
	disabled?: boolean;
}

export function ComponentPalette({
	components,
	onDragStart,
	onAddComponent,
	loading = false,
	error = null,
	onRetry,
	disabled = false,
}: Props) {
	const interactionDisabled = loading || disabled;
	return (
		<aside
			className="palette"
			aria-label="组件面板"
			aria-disabled={disabled || undefined}
		>
			<div className="palette-header">
				<h3>组件</h3>
				<span className="palette-count">{components.length}</span>
			</div>
			{loading ? (
				<div className="pipeline-palette-loading" aria-busy="true">
					<Spin size="small" />
					<span className="pipeline-loading-text">加载组件...</span>
				</div>
			) : null}
			{error ? (
				<Alert
					type="warning"
					showIcon
					message="组件加载失败"
					description={error}
					action={
						onRetry ? (
							<Button size="small" type="link" onClick={onRetry}>
								重试
							</Button>
						) : undefined
					}
					style={{ marginBottom: 8 }}
				/>
			) : null}
			{components.map((c) => (
				<button
					key={c.id}
					type="button"
					className="palette-item"
					draggable={!interactionDisabled}
					disabled={interactionDisabled}
					aria-label={`添加组件 ${c.name}`}
					title={
						disabled
							? "只读模式不可添加组件"
							: "点击添加到画布，也可以拖拽放置"
					}
					onDragStart={interactionDisabled ? undefined : (e) => onDragStart(e, c)}
					onClick={
						interactionDisabled ? undefined : () => onAddComponent?.(c)
					}
				>
					<div className="pi-content">
						<div className="pi-label">{c.name}</div>
						<div className="pi-image">{c.image}</div>
					</div>
				</button>
			))}
			{!loading && components.length === 0 ? (
				<PipelineEmptyState
					variant="palette"
					title="暂无组件"
					description={
						<>
							请先在 <Link to="/components">步骤组件</Link> 中创建。
						</>
					}
				/>
			) : null}
		</aside>
	);
}
