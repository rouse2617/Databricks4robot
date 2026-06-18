import { CopyOutlined } from "@ant-design/icons";
import { Alert, Button, Input, message, Spin, Tooltip } from "antd";
import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { componentPaletteMeta } from "./componentDisplay";
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

async function copyImageReference(image: string) {
	try {
		await navigator.clipboard.writeText(image);
		message.success("已复制镜像地址");
	} catch {
		message.error("复制失败");
	}
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
	const [query, setQuery] = useState("");
	const interactionDisabled = loading || disabled;
	const filteredComponents = useMemo(() => {
		const keyword = query.trim().toLowerCase();
		if (!keyword) return components;
		return components.filter((component) =>
			[
				component.name,
				component.image,
				component.tag,
				component.releaseLabel,
				component.sourceCommit,
				component.imageUid,
				component.id,
			]
				.filter(Boolean)
				.join(" ")
				.toLowerCase()
				.includes(keyword),
		);
	}, [components, query]);
	const hasQuery = query.trim().length > 0;

	return (
		<aside
			className="palette"
			aria-label="组件面板"
			aria-disabled={disabled || undefined}
		>
			<div className="palette-header">
				<h3>组件</h3>
				<span className="palette-count">
					{hasQuery
						? `${filteredComponents.length}/${components.length}`
						: components.length}
				</span>
			</div>
			<Input.Search
				id="pipeline-component-palette-search"
				name="componentSearch"
				allowClear
				size="small"
				value={query}
				onChange={(event) => setQuery(event.target.value)}
				placeholder="搜索名称、commit、tag"
				className="palette-search"
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
			{filteredComponents.map((c) => {
				const meta = componentPaletteMeta(c);
				return (
					<div
						key={c.id}
						className={`palette-item ${interactionDisabled ? "is-disabled" : ""}`}
						title={
							disabled
								? "只读模式不可添加组件"
								: `${meta.title}\n\n点击添加到画布，也可以拖拽放置`
						}
					>
						<button
							type="button"
							className="palette-card-main"
							draggable={!interactionDisabled}
							disabled={interactionDisabled}
							aria-label={`添加组件 ${c.name}`}
							onDragStart={
								interactionDisabled ? undefined : (e) => onDragStart(e, c)
							}
							onClick={
								interactionDisabled ? undefined : () => onAddComponent?.(c)
							}
						>
							<div className="pi-content">
								<div className="pi-label">{c.name}</div>
								{meta.subtitle ? (
									<div className="pi-subtitle">{meta.subtitle}</div>
								) : null}
							</div>
						</button>
						{c.image ? (
							<Tooltip title="复制完整镜像地址">
								<button
									type="button"
									className="palette-copy"
									aria-label={`复制组件 ${c.name} 镜像地址`}
									onClick={(event) => {
										event.stopPropagation();
										void copyImageReference(c.image);
									}}
								>
									<CopyOutlined />
								</button>
							</Tooltip>
						) : null}
					</div>
				);
			})}
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
					title="没有匹配的组件"
					description="可以按名称、commit 或 tag 搜索。"
				/>
			) : null}
		</aside>
	);
}
