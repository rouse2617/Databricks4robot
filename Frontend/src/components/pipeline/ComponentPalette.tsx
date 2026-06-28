import { CopyOutlined, DownOutlined, RightOutlined } from "@ant-design/icons";
import { Alert, Button, Input, message, Spin, Tag, Tooltip } from "antd";
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

function componentGroupKey(component: RegisteredComponent): string {
	return (
		component.componentId?.trim() ||
		component.name.trim().toLowerCase() ||
		component.id
	);
}

function componentVersionText(component: RegisteredComponent): string {
	const meta = componentPaletteMeta(component);
	return (
		component.releaseLabel ||
		component.tag ||
		component.imageUid ||
		meta.subtitle ||
		component.id
	);
}

function compareComponentVersions(
	left: RegisteredComponent,
	right: RegisteredComponent,
) {
	const leftLabel = componentVersionText(left);
	const rightLabel = componentVersionText(right);
	if (leftLabel !== rightLabel) return rightLabel.localeCompare(leftLabel);
	return left.id.localeCompare(right.id);
}

function groupComponents(components: RegisteredComponent[]) {
	const buckets = new Map<string, RegisteredComponent[]>();
	for (const component of components) {
		const key = componentGroupKey(component);
		buckets.set(key, [...(buckets.get(key) ?? []), component]);
	}
	return Array.from(buckets.values())
		.map((versions) => [...versions].sort(compareComponentVersions))
		.sort((left, right) => left[0].name.localeCompare(right[0].name));
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
	const [expandedKeys, setExpandedKeys] = useState<Set<string>>(
		() => new Set(),
	);
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
	const componentGroups = useMemo(
		() => groupComponents(filteredComponents),
		[filteredComponents],
	);
	const hasQuery = query.trim().length > 0;
	const toggleExpanded = (key: string) => {
		setExpandedKeys((current) => {
			const next = new Set(current);
			if (next.has(key)) next.delete(key);
			else next.add(key);
			return next;
		});
	};

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
						? `${componentGroups.length}/${groupComponents(components).length}`
						: componentGroups.length}
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
			{componentGroups.map((versions) => {
				const c = versions[0];
				const meta = componentPaletteMeta(c);
				const groupKey = componentGroupKey(c);
				const expanded = expandedKeys.has(groupKey);
				const hasVersions = versions.length > 1;
				return (
					<div
						key={groupKey}
						className={`palette-item ${interactionDisabled ? "is-disabled" : ""}`}
						title={
							disabled
								? "只读模式不可添加组件"
								: `${meta.title}\n\n点击添加到画布，也可以拖拽放置`
						}
					>
						{hasVersions ? (
							<button
								type="button"
								className="palette-expand"
								aria-label={`${expanded ? "收起" : "展开"}组件 ${c.name} 版本`}
								onClick={(event) => {
									event.stopPropagation();
									toggleExpanded(groupKey);
								}}
							>
								{expanded ? <DownOutlined /> : <RightOutlined />}
							</button>
						) : null}
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
								<div className="palette-version-line">
									{meta.subtitle ? (
										<span className="pi-subtitle">{meta.subtitle}</span>
									) : null}
									{hasVersions ? (
										<Tag className="palette-version-count">
											{versions.length} 个版本
										</Tag>
									) : null}
								</div>
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
						{hasVersions && expanded ? (
							<div className="palette-version-list">
								{versions.map((version, index) => {
									const versionMeta = componentPaletteMeta(version);
									const versionText = componentVersionText(version);
									return (
										<button
											key={version.id}
											type="button"
											className="palette-version-row"
											draggable={!interactionDisabled}
											disabled={interactionDisabled}
											aria-label={`添加组件 ${version.name} 版本 ${versionText}`}
											title={versionMeta.title}
											onDragStart={
												interactionDisabled
													? undefined
													: (event) => onDragStart(event, version)
											}
											onClick={
												interactionDisabled
													? undefined
													: () => onAddComponent?.(version)
											}
										>
											<span className="palette-version-row__label">
												{versionText}
											</span>
											{index === 0 ? (
												<Tag className="palette-version-default">默认</Tag>
											) : null}
											{version.sourceCommit ? (
												<span className="palette-version-row__commit">
													{version.sourceCommit.slice(0, 8)}
												</span>
											) : null}
										</button>
									);
								})}
							</div>
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
