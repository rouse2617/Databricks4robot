import { Button } from "antd";
import type { ReactNode } from "react";

export type PipelineEmptyStateVariant =
	| "canvas"
	| "config"
	| "deploy"
	| "palette";

export interface PipelineEmptyStateProps {
	variant: PipelineEmptyStateVariant;
	title: string;
	description?: ReactNode;
	hint?: string;
	action?: {
		label: string;
		onClick: () => void;
	};
	className?: string;
}

const VARIANT_CLASS: Record<PipelineEmptyStateVariant, string> = {
	canvas: "pipeline-empty pipeline-empty--canvas",
	config: "pipeline-empty pipeline-empty--config",
	deploy: "pipeline-empty pipeline-empty--deploy",
	palette: "pipeline-empty pipeline-empty--palette",
};

export function PipelineEmptyState({
	variant,
	title,
	description,
	hint,
	action,
	className,
}: PipelineEmptyStateProps) {
	return (
		<div
			className={[VARIANT_CLASS[variant], className].filter(Boolean).join(" ")}
			role="status"
			aria-live="polite"
		>
			<div className="pipeline-empty__title">{title}</div>
			{description ? (
				<div className="pipeline-empty__description">{description}</div>
			) : null}
			{hint ? <p className="pipeline-empty__hint">{hint}</p> : null}
			{action ? (
				<Button size="small" type="link" onClick={action.onClick}>
					{action.label}
				</Button>
			) : null}
		</div>
	);
}
