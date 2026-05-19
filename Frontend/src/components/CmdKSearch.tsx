// ─── CmdKSearch — Global Cmd+K / Ctrl+K search for assets, MCAP files, deliveries ───

import { SearchOutlined } from "@ant-design/icons";
import { Input, List, Modal, Typography } from "antd";
import type { InputRef } from "antd/es/input";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { searchApi } from "../api/search";

const { Text } = Typography;

interface SearchResultGroup {
	category: string;
	items: SearchResultItem[];
}

interface SearchResultItem {
	id: string;
	label: string;
	subtitle?: string;
	onSelect: () => void;
}

const MAX_PER_GROUP = 5;

export default function CmdKSearch() {
	const navigate = useNavigate();
	const [open, setOpen] = useState(false);
	const [query, setQuery] = useState("");
	const [loading, setLoading] = useState(false);
	const [groups, setGroups] = useState<SearchResultGroup[]>([]);
	const inputRef = useRef<InputRef>(null);

	const isAssetId = useMemo(
		() => /^[A-Za-z0-9]{8}$/.test(query.trim()),
		[query],
	);
	const isUUID = useMemo(
		() =>
			/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(
				query.trim(),
			),
		[query],
	);

	const handleKeyDown = useCallback((e: KeyboardEvent) => {
		if ((e.metaKey || e.ctrlKey) && e.key === "k") {
			e.preventDefault();
			setOpen((prev) => !prev);
		}
	}, []);

	useEffect(() => {
		window.addEventListener("keydown", handleKeyDown);
		return () => window.removeEventListener("keydown", handleKeyDown);
	}, [handleKeyDown]);

	useEffect(() => {
		if (!open) return;
		const timer = setTimeout(() => inputRef.current?.focus(), 100);
		return () => clearTimeout(timer);
	}, [open]);

	const doSearch = useCallback(async () => {
		const q = query.trim();
		if (!q || q.length < 2) {
			setGroups([]);
			return;
		}

		setLoading(true);
		const resultGroups: SearchResultGroup[] = [];

		try {
			// Direct asset ID match — navigate immediately without search
			if (isAssetId) {
				resultGroups.push({
					category: "直接匹配",
					items: [
						{
							id: `asset:${q}`,
							label: `资产 ${q}`,
							subtitle: "Asset ID",
							onSelect: () => {
								navigate(`/assets/${q}`);
								setOpen(false);
								setQuery("");
							},
						},
						{
							id: `mcap:${q}`,
							label: `MCAP ${q}`,
							subtitle: "MCAP File ID",
							onSelect: () => {
								navigate(`/mcap-files/${q}`);
								setOpen(false);
								setQuery("");
							},
						},
					],
				});
				setGroups(resultGroups);
				setLoading(false);
				return;
			}

			if (isUUID) {
				const group: SearchResultGroup = {
					category: "直接匹配",
					items: [
						{
							id: `delivery:${q}`,
							label: `交付 ${q.slice(0, 8)}...`,
							subtitle: "Delivery ID (UUID)",
							onSelect: () => {
								navigate(`/deliveries/${q}`);
								setOpen(false);
								setQuery("");
							},
						},
					],
				};
				setGroups([group]);
				setLoading(false);
				return;
			}

			// ES search
			const [assetRes] = await Promise.allSettled([
				searchApi.searchAssets({ q, page_size: MAX_PER_GROUP }),
			]);

			if (assetRes.status === "fulfilled" && assetRes.value.items.length > 0) {
				resultGroups.push({
					category: `资产 (${assetRes.value.total})`,
					items: assetRes.value.items.map((a) => ({
						id: `asset:${a.asset_id}`,
						label: a.asset_id,
						subtitle: [a.asset_type, a.lifecycle_state, a.env]
							.filter(Boolean)
							.join(" · "),
						onSelect: () => {
							navigate(`/assets/${a.asset_id}`);
							setOpen(false);
							setQuery("");
						},
					})),
				});
			}
		} catch {
			// best-effort
		} finally {
			setLoading(false);
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [query, isAssetId, isUUID, navigate]);

	useEffect(() => {
		doSearch();
	}, [doSearch]);

	const handleClose = () => {
		setOpen(false);
		setQuery("");
		setGroups([]);
	};

	return (
		<Modal
			open={open}
			onCancel={handleClose}
			footer={null}
			width={560}
			title={null}
			centered
			closable
			maskClosable
			styles={{ body: { padding: 0 } }}
		>
			<div style={{ padding: "12px 16px 0 16px" }}>
				<Input
					ref={inputRef}
					prefix={<SearchOutlined style={{ color: "#999" }} />}
					placeholder="搜索资产、MCAP、交付…  (8位ID精确匹配)"
					value={query}
					onChange={(e) => setQuery(e.target.value)}
					variant="borderless"
					size="large"
					style={{ fontSize: 16 }}
				/>
			</div>

			<div
				style={{
					borderTop: groups.length > 0 ? "1px solid #f0f0f0" : "none",
					marginTop: loading ? 0 : 8,
				}}
			>
				{loading && (
					<div
						style={{
							textAlign: "center",
							padding: 20,
							color: "#999",
							fontSize: 13,
						}}
					>
						搜索中…
					</div>
				)}
				{!loading && query.trim().length > 0 && groups.length === 0 && (
					<div
						style={{
							textAlign: "center",
							padding: 20,
							color: "#999",
							fontSize: 13,
						}}
					>
						无结果
					</div>
				)}
				{!loading &&
					groups.map((g) => (
						<div key={g.category}>
							<div
								style={{
									padding: "8px 16px 4px 16px",
									color: "#999",
									fontSize: 11,
									fontWeight: 600,
									textTransform: "uppercase",
									letterSpacing: "0.5px",
								}}
							>
								{g.category}
							</div>
							<List
								dataSource={g.items}
								renderItem={(item) => (
									<List.Item
										key={item.id}
										style={{
											padding: "8px 16px",
											cursor: "pointer",
										}}
										onClick={item.onSelect}
										onMouseEnter={(e) => {
											(e.currentTarget as HTMLElement).style.background =
												"#f5f5f5";
										}}
										onMouseLeave={(e) => {
											(e.currentTarget as HTMLElement).style.background =
												"transparent";
										}}
									>
										<div>
											<Text
												strong
												style={{ fontFamily: "monospace", fontSize: 13 }}
											>
												{item.label}
											</Text>
											{item.subtitle && (
												<>
													<br />
													<Text type="secondary" style={{ fontSize: 12 }}>
														{item.subtitle}
													</Text>
												</>
											)}
										</div>
									</List.Item>
								)}
							/>
						</div>
					))}
			</div>
		</Modal>
	);
}
