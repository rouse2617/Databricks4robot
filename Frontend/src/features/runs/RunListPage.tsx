import { ExecutionRecordsPanel } from "../../pages/ExecutionRecordsPanel";

export function RunListPage() {
	return (
		<div className="pipeline-tab-content pipeline-tab-content--panel pipeline-tab-content--executions">
			<ExecutionRecordsPanel active />
		</div>
	);
}
