import { Table, Tag } from "antd";
import { useEffect, useState } from "react";
import { getRunPods, type RunPodItem } from "../api/podsApi";

export function RunPodsPanel({ runId }: { runId: string }) {
	const [items, setItems] = useState<RunPodItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		let alive = true;
		setLoading(true);
		getRunPods(runId)
			.then((response) => {
				if (!alive) return;
				setItems(response.items);
				setError(null);
			})
			.catch((err) => {
				if (!alive) return;
				setError(String(err));
			})
			.finally(() => {
				if (alive) setLoading(false);
			});
		return () => {
			alive = false;
		};
	}, [runId]);

	if (error) {
		return <div>{error}</div>;
	}

	return (
		<Table
			rowKey={(row) => row.podName}
			loading={loading}
			dataSource={items}
			pagination={false}
			columns={[
				{ title: "Pod", dataIndex: "podName", key: "podName" },
				{ title: "节点", dataIndex: "displayName", key: "displayName" },
				{
					title: "状态",
					dataIndex: "phase",
					key: "phase",
					render: (phase: string) => <Tag>{phase}</Tag>,
				},
				{ title: "Pod IP", dataIndex: "podIp", key: "podIp" },
				{ title: "重启", dataIndex: "restartCount", key: "restartCount" },
			]}
		/>
	);
}
