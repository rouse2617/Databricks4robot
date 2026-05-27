import { Button, Tag, message } from "antd";
import { useEffect, useState, useCallback } from "react";
import {
	listDeployments,
	listPipelines,
	deletePipeline,
	deleteDeployment,
	deployTemplate,
	type PipelineTemplate,
	type Deployment,
} from "../../api/pipelineApi";
import { ReloadOutlined, DeleteOutlined, PlayCircleOutlined } from "@ant-design/icons";

const STATUS_COLORS: Record<string, string> = {
	Succeeded: "success",
	Running: "processing",
	Pending: "warning",
	Failed: "error",
	Error: "error",
};

export function DeployPanel() {
	const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
	const [deployments, setDeployments] = useState<Deployment[]>([]);
	const [loading, setLoading] = useState(false);

	const refresh = useCallback(async () => {
		setLoading(true);
		try {
			const [d, t] = await Promise.all([listDeployments(), listPipelines()]);
			setDeployments(d);
			setTemplates(t);
		} catch {
			/* server not available */
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		refresh();
	}, [refresh]);

	const handleDeployTemplate = async (id: string) => {
		try {
			await deployTemplate(id);
			refresh();
		} catch (err) {
			console.error(err);
		}
	};

	const handleDeleteDeployment = async (id: string) => {
		try {
			await deleteDeployment(id);
			message.success("已删除运行记录");
			refresh();
		} catch (err) {
			message.error("删除失败: " + String(err));
		}
	};

	const handleDeleteTemplate = async (id: string) => {
		try {
			await deletePipeline(id);
			message.success("已删除流水线模板");
			refresh();
		} catch (err) {
			message.error("删除失败: " + String(err));
		}
	};

	return (
		<div className="deploy-panel">
			<div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 16 }}>
				<h3>运行记录</h3>
				<Button size="small" icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
					刷新
				</Button>
			</div>

			<div className="deploy-section-title">
				已保存的流水线
				<span className="count">{templates.length}</span>
			</div>
			<div className="deploy-section">
				{templates.length === 0 ? (
					<div className="dep-empty">暂无已保存的流水线模板</div>
				) : (
					templates.map((t) => (
						<div key={t.id} className="dep-card">
							<div className="dep-card-info">
								<div className="dep-card-name">{t.name}</div>
								<div className="dep-card-meta">
									<span>{t.nodeCount} 个节点</span>
									<span className="dot">•</span>
									<span>{new Date(t.createdAt).toLocaleString()}</span>
								</div>
							</div>
							<div className="deploy-btn-list">
								<Button
									size="small"
									type="primary"
									icon={<PlayCircleOutlined />}
									onClick={() => handleDeployTemplate(t.id)}
								>
									运行
								</Button>
								<Button
									size="small"
									danger
									icon={<DeleteOutlined />}
									onClick={() => handleDeleteTemplate(t.id)}
								/>
							</div>
						</div>
					))
				)}
			</div>

			<div className="deploy-section-title" style={{ marginTop: 20 }}>
				运行历史
				<span className="count">{deployments.length}</span>
			</div>
			<div className="deploy-section">
				{deployments.length === 0 ? (
					<div className="dep-empty">暂无运行记录</div>
				) : (
					deployments.map((d) => (
						<div key={d.id} className="dep-card">
							<div className="dep-card-info">
								<div className="dep-card-name">{d.pipelineName}</div>
								<div className="dep-card-meta">
									<Tag color={STATUS_COLORS[d.status] || "default"}>{d.status}</Tag>
									<span>{d.nodeCount} 个节点</span>
									<span className="dot">•</span>
									<span>{new Date(d.createdAt).toLocaleString()}</span>
									{d.finishedAt && (
										<>
											<span className="dot">•</span>
											<span>完成: {new Date(d.finishedAt).toLocaleString()}</span>
										</>
									)}
								</div>
							</div>
							<div className="deploy-btn-list">
								<Button
									size="small"
									danger
									icon={<DeleteOutlined />}
									onClick={() => handleDeleteDeployment(d.id)}
								/>
							</div>
						</div>
					))
				)}
			</div>
		</div>
	);
}
