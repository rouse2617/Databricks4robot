## 2026-06-05 - Chrome MCP fallback

- **Context**: This change touches `Frontend/`, so dev UI verification is required after deploying the Cloudflare Worker.
- **Decision**: Chrome DevTools MCP returned `Transport closed` after clearing stale MCP/profile processes. I used the available agentyc browser MCP to verify the deployed dev page instead.
- **Alternatives**: Stop and ask the user to manually test, or rely only on Vitest/build checks.
- **Rationale**: agentyc provided browser navigation, DOM inspection, form input, page search, console log inspection, and was enough to validate the component palette search behavior without modifying product data.

## 2026-06-05 - Dev deploy verification

- **Context**: Frontend dev was deployed to Cloudflare Worker `cyber-databrew-dev`.
- **Decision**: Verified `https://cyber-databrew-dev.cyberorigin.ai/pipeline?tab=design` with agentyc MCP.
- **Alternatives**: Use local Vite only, which would not prove the deployed Worker bundle.
- **Rationale**: The deployed page showed the new component search input. Searching `gpu` reduced the palette to GPU-related components, searching a non-existent term showed the search-specific empty state and clear action, and searching `head` reduced the palette to `head-track-pycuvslam`. Console logs were empty.
