import type { RegisteredComponent } from "./types";

export interface ImageIdentity {
	repository: string;
	version: string;
	detail: string;
}

export function imageIdentity(image: string): ImageIdentity {
	const value = image.trim();
	if (!value) return { repository: "", version: "", detail: "" };

	const leaf = value.split("/").pop() ?? value;
	const [repoWithMaybeTag, digest] = leaf.split("@sha256:");
	const lastColon = repoWithMaybeTag.lastIndexOf(":");
	const hasTag = lastColon > 0;
	const repository = hasTag
		? repoWithMaybeTag.slice(0, lastColon)
		: repoWithMaybeTag;
	const imageTag = hasTag ? repoWithMaybeTag.slice(lastColon + 1) : "";
	const shortDigest = digest ? `sha256:${digest.slice(0, 8)}` : "";

	return {
		repository: repository || leaf,
		version: imageTag || shortDigest,
		detail: value,
	};
}

export function componentPaletteMeta(component: RegisteredComponent) {
	const image = imageIdentity(component.image);
	const version =
		component.releaseLabel ||
		component.tag ||
		image.version ||
		component.imageUid;
	const subtitle = version || image.repository || component.image;

	return {
		subtitle,
		title: [
			component.name,
			image.repository ? `image name: ${image.repository}` : "",
			version ? `version: ${version}` : "",
			component.sourceCommit ? `commit: ${component.sourceCommit}` : "",
			component.image ? `image: ${image.detail}` : "",
		]
			.filter(Boolean)
			.join("\n"),
	};
}
