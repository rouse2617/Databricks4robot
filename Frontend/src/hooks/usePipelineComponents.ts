import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from "react";
import { listComponents, type PipelineComponentAPI } from "../api/pipelineComponentApi";
import type { RegisteredComponent } from "../components/pipeline/types";

const STORAGE_KEY = "databrew-components";

function loadComponentsFromStorage(): RegisteredComponent[] {
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (raw) return JSON.parse(raw) as RegisteredComponent[];
	} catch {
		/* ignore */
	}
	return [];
}

function saveComponentsToStorage(comps: RegisteredComponent[]) {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(comps));
}

export type UsePipelineComponentsResult = {
	components: RegisteredComponent[];
	loading: boolean;
	error: string | null;
	reload: () => void;
	setComponents: Dispatch<SetStateAction<RegisteredComponent[]>>;
};

export function usePipelineComponents(
	mapApi: (api: PipelineComponentAPI) => RegisteredComponent,
	dedupe: (comps: RegisteredComponent[]) => RegisteredComponent[],
): UsePipelineComponentsResult {
	const [components, setComponents] = useState<RegisteredComponent[]>(
		loadComponentsFromStorage,
	);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const fetchComponents = useCallback(() => {
		setLoading(true);
		setError(null);
		listComponents()
			.then((res) => {
				const mapped = dedupe((res.items ?? []).map(mapApi));
				if (mapped.length > 0) {
					setComponents(mapped);
					saveComponentsToStorage(mapped);
				}
			})
			.catch((err) => {
				setError(String(err));
			})
			.finally(() => {
				setLoading(false);
			});
	}, [dedupe, mapApi]);

	useEffect(() => {
		fetchComponents();
	}, [fetchComponents]);

	useEffect(() => {
		saveComponentsToStorage(components);
	}, [components]);

	return {
		components,
		loading,
		error,
		reload: fetchComponents,
		setComponents,
	};
}
