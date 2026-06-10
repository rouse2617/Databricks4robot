import { splitLinkifiedText } from "../../lib/workflow-utils";

export function LinkifiedText({ text }: { text: string }) {
	return (
		<span>
			{splitLinkifiedText(text).map((part) =>
				part.type === "link" ? (
					<a
						key={`link-${part.start}-${part.text}`}
						href={part.href}
						target="_blank"
						rel="noreferrer"
						onClick={(event) => event.stopPropagation()}
					>
						{part.text}
					</a>
				) : (
					<span key={`text-${part.start}`}>{part.text}</span>
				),
			)}
		</span>
	);
}
