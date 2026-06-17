import { useReducer } from "react";
import {
	designerReducer,
	type DesignerAction,
	type DesignerState,
	initialDesignerState,
} from "./designerReducer";

export function useDesignerReducer(
	initialState: DesignerState = initialDesignerState,
) {
	return useReducer(designerReducer, initialState);
}

export type { DesignerAction, DesignerState };
export { designerReducer, initialDesignerState };
