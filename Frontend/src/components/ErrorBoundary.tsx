import { Button, Result } from "antd";
import type { ErrorInfo, ReactNode } from "react";
import { Component } from "react";

interface Props {
	children: ReactNode;
}

interface State {
	hasError: boolean;
	error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
	constructor(props: Props) {
		super(props);
		this.state = { hasError: false, error: null };
	}

	static getDerivedStateFromError(error: Error): State {
		return { hasError: true, error };
	}

	componentDidCatch(error: Error, errorInfo: ErrorInfo) {
		console.error("ErrorBoundary caught:", error, errorInfo);
	}

	render() {
		if (this.state.hasError) {
			return (
				<Result
					status="error"
					title="页面出错了"
					subTitle={this.state.error?.message ?? "未知错误"}
					extra={
						<Button type="primary" onClick={() => window.location.reload()}>
							刷新页面
						</Button>
					}
				/>
			);
		}
		return this.props.children;
	}
}
