import { Spin } from "antd";
import { useEffect, useState } from "react";
import { Navigate, useParams } from "react-router-dom";
import WorkflowDetailPage from "../../pages/WorkflowDetailPage";
import { getRun, getRunByWorkflowName } from "./api/runsApi";

export { RunListPage } from "./RunListPage";

export function RunInspectorPage() {
	const { runId } = useParams<{ runId: string }>();
	const [workflowName, setWorkflowName] = useState<string | null>(null);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		if (!runId) return;
		let alive = true;
		setWorkflowName(null);
		setError(null);
		getRun(runId)
			.then((run) => {
				if (!alive) return;
				setWorkflowName(run.runtime.resourceName);
			})
			.catch((err) => {
				if (!alive) return;
				setError(String(err));
			});
		return () => {
			alive = false;
		};
	}, [runId]);

	if (error) {
		return <div className="pipeline-tab-content">{error}</div>;
	}
	if (!runId || workflowName === null) {
		return (
			<div
				className="flex items-center justify-center"
				style={{ height: "60vh" }}
			>
				<Spin size="large" />
			</div>
		);
	}

	return <WorkflowDetailPage legacyRoute={false} />;
}

export function RunRedirectFromWorkflowName() {
	const { name } = useParams<{ name: string }>();
	const [targetRunId, setTargetRunId] = useState<string | null>(null);
	const [missing, setMissing] = useState(false);

	useEffect(() => {
		if (!name) return;
		let alive = true;
		getRunByWorkflowName(name)
			.then((run) => {
				if (!alive) return;
				setTargetRunId(run.id);
			})
			.catch(() => {
				if (!alive) return;
				setMissing(true);
			});
		return () => {
			alive = false;
		};
	}, [name]);

	if (missing && name) {
		return <WorkflowDetailPage legacyRoute />;
	}
	if (!targetRunId) {
		return (
			<div
				className="flex items-center justify-center"
				style={{ height: "60vh" }}
			>
				<Spin size="large" />
			</div>
		);
	}
	return <Navigate to={`/runs/${encodeURIComponent(targetRunId)}`} replace />;
}
