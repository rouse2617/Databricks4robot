import { Alert, Button, Input, Spin } from "antd";
import { useMemo, useState } from "react";
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
}

export function ComponentPalette({
	components,
	onDragStart,
	onAddComponent,
	loading = false,
	error = null,
	onRetry,
}: Props) {
	const [search, setSearch] = useState("");
	const filtered = useMemo(() => {
		const q = search.toLowerCase().trim();
		if (!q) return components;
		return components.filter(
			(c) =>
				c.name.toLowerCase().includes(q) ||
				(c.image || "").toLowerCase().includes(q),
		);
	}, [components, search]);

	return (
		<aside className="palette" aria-label="组件面板">
			<div className="palette-header">
				<h3>组件</h3>
				<span className="palette-count">{filtered.length}</span>
			</div>
			<Input.Search
				placeholder="搜索组件..."
				value={search}
				onChange={(e) => setSearch(e.target.value)}
				allowClear
				style={{ marginBottom: 8 }}
			/>
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
					description={search ? undefined : error}
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
			{filtered.map((c) => (
				<button
					key={c.id}
					type="button"
					className="palette-item"
					draggable={!loading}
					aria-label={`添加组件 ${c.name}`}
					title="点击添加到画布，也可以拖拽放置"
					onDragStart={(e) => onDragStart(e, c)}
					onClick={() => onAddComponent?.(c)}
				>
					<div className="pi-content">
						<div className="pi-label">{c.name}</div>
						<div className="pi-image">{c.image}</div>
					</div>
				</button>
			))}
			{!loading && filtered.length === 0 && search ? (
				<PipelineEmptyState
					variant="palette"
					title={search ? "无匹配组件" : "暂无组件"}
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
