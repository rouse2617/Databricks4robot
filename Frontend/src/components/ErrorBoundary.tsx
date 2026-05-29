import { Button, Result } from "antd";
import type { ErrorInfo, ReactNode } from "react";
import { Component } from "react";

interface Props {
	children: ReactNode;
	title?: string;
	subTitle?: string;
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

	handleRetry = () => {
		this.setState({ hasError: false, error: null });
	};

	render() {
		if (this.state.hasError) {
			return (
				<Result
					status="error"
					title={this.props.title ?? "页面出错了"}
					subTitle={
						this.state.error?.message ?? this.props.subTitle ?? "未知错误"
					}
					extra={[
						<Button key="retry" type="primary" onClick={this.handleRetry}>
							重试
						</Button>,
						<Button key="reload" onClick={() => window.location.reload()}>
							刷新页面
						</Button>,
					]}
				/>
			);
		}
		return this.props.children;
	}
}
