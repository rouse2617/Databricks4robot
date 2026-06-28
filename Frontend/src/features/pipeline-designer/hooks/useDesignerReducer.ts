import { useReducer } from "react";
import {
	type DesignerAction,
	type DesignerState,
	designerReducer,
	initialDesignerState,
} from "./designerReducer";

export function useDesignerReducer(
	initialState: DesignerState = initialDesignerState,
) {
	return useReducer(designerReducer, initialState);
}

export type { DesignerAction, DesignerState };
export { designerReducer, initialDesignerState };
