const REACT_FLOW_TYPES_WARNING =
	"[React Flow]: It looks like you've created a new nodeTypes or edgeTypes object.";

let installed = false;

export function installConsoleWarningFilter() {
	if (installed) return;
	installed = true;

	const originalWarn = console.warn.bind(console);
	console.warn = (...args: unknown[]) => {
		const firstArg = args[0];
		if (
			typeof firstArg === "string" &&
			firstArg.startsWith(REACT_FLOW_TYPES_WARNING)
		) {
			return;
		}
		originalWarn(...args);
	};
}
