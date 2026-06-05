import { SearchOutlined } from "@ant-design/icons";
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
	const [query, setQuery] = useState("");
	const normalizedQuery = query.trim().toLowerCase();
	const filteredComponents = useMemo(() => {
		if (!normalizedQuery) return components;
		return components.filter((component) => {
			const haystack = [
				component.name,
				component.image,
				component.tag,
				component.computeTier,
			]
				.filter(Boolean)
				.join(" ")
				.toLowerCase();
			return haystack.includes(normalizedQuery);
		});
	}, [components, normalizedQuery]);
	const hasSearch = normalizedQuery.length > 0;

	return (
		<aside className="palette" aria-label="组件面板">
			<div className="palette-header">
				<h4>组件</h4>
				<span className="palette-count">
					{filteredComponents.length}
					{hasSearch ? ` / ${components.length}` : ""}
				</span>
			</div>
			<Input
				allowClear
				size="small"
				prefix={<SearchOutlined />}
				placeholder="搜索组件名称、镜像、标签或算力层"
				value={query}
				onChange={(event) => setQuery(event.target.value)}
				aria-label="搜索组件"
			/>
			{loading ? (
				<div className="pipeline-palette-loading" aria-busy="true">
					<Spin size="small" tip="加载组件..." />
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
			{filteredComponents.map((c) => (
				<button
					key={c.id}
					type="button"
					className="palette-item"
					draggable={!loading}
					aria-label={`拖入组件 ${c.name}`}
					onDragStart={(e) => onDragStart(e, c)}
					onClick={() => onAddComponent?.(c)}
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
			{!loading && components.length > 0 && filteredComponents.length === 0 ? (
				<PipelineEmptyState
					variant="palette"
					title="未找到匹配组件"
					description="换个关键词试试，或清空搜索后查看全部组件。"
					action={{
						label: "清空搜索",
						onClick: () => setQuery(""),
					}}
				/>
			) : null}
		</aside>
	);
}
