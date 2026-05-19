// Shared truncated-id link used in tables across pages.
//
// Replaces ad-hoc `<button class="font-mono text-xs link-like-button">`
// snippets that were duplicated in DashboardPage / SavedQueriesPage / etc.

import { Tooltip } from "antd";
import { useNavigate } from "react-router-dom";
import { navigateToAssetDetail } from "../../lib/assets/assetWorkbenchNavigation";

export interface AssetIdLinkProps {
	id: string;
	/** Number of leading characters to keep before the ellipsis. */
	keep?: number;
	/** Optional click handler; defaults to `navigateToAssetDetail`. */
	onClick?: (id: string) => void;
	/** Show full id as native tooltip on hover. */
	tooltip?: boolean;
}

export default function AssetIdLink({
	id,
	keep = 8,
	onClick,
	tooltip = true,
}: AssetIdLinkProps) {
	const navigate = useNavigate();
	const display = id.length > keep ? `${id.slice(0, keep)}…` : id;

	const button = (
		<button
			type="button"
			className="font-mono text-xs cursor-pointer link-like-button"
			aria-label={`Asset ${id}`}
			onClick={(e) => {
				e.stopPropagation();
				if (onClick) onClick(id);
				else navigateToAssetDetail(navigate, id);
			}}
		>
			{display}
		</button>
	);

	if (!tooltip) return button;
	return <Tooltip title={id}>{button}</Tooltip>;
}
