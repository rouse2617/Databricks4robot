import type { TableRowSelection } from "antd/es/table/interface";

export const TABLE_SELECT_ALL_TITLE = "全选";

/** Ant Design Table rowSelection with localized select-all column header. */
export function withSelectAllColumn<T>(
	selection: TableRowSelection<T>,
): TableRowSelection<T> {
	if (selection.columnTitle != null) {
		return selection;
	}
	return {
		...selection,
		columnTitle: TABLE_SELECT_ALL_TITLE as TableRowSelection<T>["columnTitle"],
	};
}
