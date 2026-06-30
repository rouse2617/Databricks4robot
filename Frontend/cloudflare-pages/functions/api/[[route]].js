// Proxy /api/* to Cloud Run backend (server-side, no CORS issues)
export async function onRequest(context) {
	const { request } = context;
	const url = new URL(request.url);
	const backendHost = "cyber-databrew-backend-prod-wtttm6suaq-uc.a.run.app";
	const targetUrl = `https://${backendHost}${url.pathname}${url.search}`;

	const modifiedRequest = new Request(targetUrl, {
		method: request.method,
		headers: request.headers,
		body:
			request.method !== "GET" && request.method !== "HEAD"
				? request.body
				: undefined,
	});
	modifiedRequest.headers.set("Host", backendHost);

	try {
		const response = await fetch(modifiedRequest);
		const responseHeaders = new Headers(response.headers);
		responseHeaders.delete("alt-svc");
		responseHeaders.delete("server");
		// Cloudflare Response: null-body statuses (204, 304) cannot have a body
		const isNullBodyStatus =
			response.status === 204 || response.status === 304;
		return new Response(isNullBodyStatus ? null : response.body, {
			status: response.status,
			statusText: response.statusText,
			headers: responseHeaders,
		});
	} catch (err) {
		return new Response(
			JSON.stringify({ error: "backend unavailable", detail: err.message }),
			{ status: 502, headers: { "Content-Type": "application/json" } },
		);
	}
}
