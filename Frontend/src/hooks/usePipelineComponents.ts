import {
	type Dispatch,
	type SetStateAction,
	useCallback,
	useEffect,
	useState,
} from "react";
import {
	listComponentReleases,
	listComponents,
	type PipelineComponentAPI,
	type PipelineComponentReleaseAPI,
} from "../api/pipelineComponentApi";
import type { RegisteredComponent } from "../components/pipeline/types";

const STORAGE_KEY = "databrew-components";

function hasLocalStorage(): boolean {
	return typeof localStorage !== "undefined";
}

function loadComponentsFromStorage(): RegisteredComponent[] {
	if (!hasLocalStorage()) return [];
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (raw) return JSON.parse(raw) as RegisteredComponent[];
	} catch {
		/* ignore */
	}
	return [];
}

function saveComponentsToStorage(comps: RegisteredComponent[]) {
	if (!hasLocalStorage()) return;
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(comps));
	} catch {
		/* ignore */
	}
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
	mapRelease?: (api: PipelineComponentReleaseAPI) => RegisteredComponent,
): UsePipelineComponentsResult {
	const [components, setComponents] = useState<RegisteredComponent[]>(
		loadComponentsFromStorage,
	);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const fetchComponents = useCallback(() => {
		setLoading(true);
		setError(null);
		Promise.all([
			listComponents(),
			mapRelease
				? listComponentReleases({ selectable: true }).catch(() => ({
						items: [] as PipelineComponentReleaseAPI[],
					}))
				: Promise.resolve({ items: [] as PipelineComponentReleaseAPI[] }),
		])
			.then(([componentRes, releaseRes]) => {
				const releaseComponents = mapRelease
					? (releaseRes.items ?? []).map(mapRelease)
					: [];
				const releaseComponentIds = new Set(
					releaseComponents
						.map((component) => component.componentId?.trim())
						.filter((value): value is string => Boolean(value)),
				);
				const legacyComponents = (componentRes.items ?? [])
					.map(mapApi)
					.filter(
						(component) =>
							!releaseComponentIds.has(component.componentId?.trim() ?? ""),
					);
				const mapped = dedupe([...releaseComponents, ...legacyComponents]);
				setComponents(mapped);
				saveComponentsToStorage(mapped);
			})
			.catch((err) => {
				setError(String(err));
			})
			.finally(() => {
				setLoading(false);
			});
	}, [dedupe, mapApi, mapRelease]);

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
