// Proxy /api/* requests to Cloud Run backend.
// Routes to dev backend when served from a *-dev.* hostname
// (e.g. cyber-databrew-dev.cyberorigin.ai); otherwise prod.
const PREVIEW_ORIGIN_HOST = "api-cyber-databrew-dev.cyberorigin.ai";
const PREVIEW_RESOLVE_OVERRIDE_HOST =
  "preview-origin-cyber-databrew-dev.cyberorigin.ai";
const PREVIEW_PATH_RE = /^\/preview\/([0-9a-f]{7,40})(?:\/|$)/i;

export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    const previewMatch = url.pathname.match(PREVIEW_PATH_RE);
    if (previewMatch) {
      return proxyToHost(request, PREVIEW_ORIGIN_HOST, {
        resolveOverride: PREVIEW_RESOLVE_OVERRIDE_HOST,
        headers: {
          "X-Preview-Id": previewMatch[1],
          "X-Forwarded-Host": url.hostname,
          "X-Forwarded-Proto": url.protocol.replace(":", ""),
        },
        error: "preview origin unavailable",
      });
    }

    if (url.pathname.startsWith("/api/")) {
      const isDev = url.hostname.includes("-dev.");
      const backendHost = isDev
        ? "cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app"
        : "cyber-databrew-backend-prod-wtttm6suaq-uc.a.run.app";
      return proxyToHost(request, backendHost, {
        error: "backend unavailable",
      });
    }

    // Serve Docusaurus docs under /doc/ with SPA fallback
    if (url.pathname.startsWith("/doc/")) {
      let response = await env.ASSETS.fetch(request);
      if (response.status === 404) {
        response = await env.ASSETS.fetch(
          new Request(new URL("/doc/index.html", request.url))
        );
      }
      return response;
    }

    // Try static assets, fall back to index.html for SPA routing. Hashed chunks
    // must not fall back to index.html, otherwise old tabs receive HTML for a
    // JavaScript module and get stuck in the app error boundary.
    const response = await env.ASSETS.fetch(request);
    if (response.status === 404) {
      if (isStaticAssetPath(url.pathname)) {
        return response;
      }
      return env.ASSETS.fetch(new Request(new URL("/index.html", request.url)));
    }
    return response;
  },
};

async function proxyToHost(request, targetHost, options = {}) {
  const url = new URL(request.url);
  const targetUrl = "https://" + targetHost + url.pathname + url.search;
  const headers = new Headers(request.headers);
  headers.set("Host", targetHost);

  for (const [key, value] of Object.entries(options.headers || {})) {
    headers.set(key, value);
  }

  try {
    return await fetch(targetUrl, {
      method: request.method,
      headers,
      cf: options.resolveOverride
        ? { resolveOverride: options.resolveOverride }
        : undefined,
      body: request.method !== "GET" && request.method !== "HEAD"
        ? request.body
        : undefined,
    });
  } catch (err) {
    return new Response(
      JSON.stringify({
        error: options.error || "origin unavailable",
        detail: err.message,
      }),
      { status: 502, headers: { "Content-Type": "application/json" } }
    );
  }
}

function isStaticAssetPath(pathname) {
  return pathname.startsWith("/assets/") && /\.[A-Za-z0-9]+$/.test(pathname);
}
