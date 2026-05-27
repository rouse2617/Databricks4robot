// Proxy /api/* requests to Cloud Run backend
export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    if (url.pathname.startsWith("/api/")) {
      const backendHost = "cyber-databrew-backend-prod-wtttm6suaq-uc.a.run.app";
      const targetUrl = "https://" + backendHost + url.pathname + url.search;

      const headers = new Headers(request.headers);
      headers.set("Host", backendHost);

      try {
        return await fetch(targetUrl, {
          method: request.method,
          headers,
          body: request.method !== "GET" && request.method !== "HEAD"
            ? request.body
            : undefined,
        });
      } catch (err) {
        return new Response(
          JSON.stringify({ error: "backend unavailable", detail: err.message }),
          { status: 502, headers: { "Content-Type": "application/json" } }
        );
      }
    }

    // Try static assets, fall back to index.html for SPA routing
    const response = await env.ASSETS.fetch(request);
    if (response.status === 404) {
      return env.ASSETS.fetch(new Request(new URL("/index.html", request.url)));
    }
    return response;
  },
};
