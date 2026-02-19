import type { Plugin, ViteDevServer, Connect } from "vite";

// Store the current request's cookies for use in transformIndexHtml
let currentRequestCookies: string | undefined;

export function gappPreloadPlugin(options?: {
  serverUrl?: string;
  preloadPath?: string;
}): Plugin {
  const serverUrl = options?.serverUrl ?? "http://localhost:8080";
  const preloadPath = options?.preloadPath ?? "/__preload";

  return {
    name: "gapp-preload",

    configureServer(server: ViteDevServer) {
      server.httpServer?.once("listening", () => {
        console.log(
          "[gapp-preload] Plugin active, fetching preloads from",
          serverUrl
        );
      });

      // Middleware to capture cookies from incoming requests
      server.middlewares.use((req: Connect.IncomingMessage, _res, next) => {
        currentRequestCookies = req.headers.cookie;
        next();
      });
    },

    transformIndexHtml: {
      order: "pre",
      async handler(html, ctx) {
        const originalUrl = (ctx as any).originalUrl as string | undefined;
        const path = originalUrl
          ? new URL(originalUrl, "http://localhost").pathname
          : "/";

        try {
          const url = `${serverUrl}${preloadPath}?path=${encodeURIComponent(
            path
          )}`;

          console.log(
            `[gapp-preload] Fetching preloads for path="${path}" (originalUrl="${originalUrl}"), cookies: ${
              currentRequestCookies ? "present" : "none"
            }`
          );

          const res = await fetch(url, {
            headers: currentRequestCookies
              ? { Cookie: currentRequestCookies }
              : {},
            signal: AbortSignal.timeout(2000),
          });

          if (!res.ok) {
            console.warn(
              `[gapp-preload] Server returned ${res.status} for ${path}`
            );
            return html;
          }

          const preloaded = await res.json();
          const methods = Object.keys(preloaded);

          if (methods.length > 0) {
            console.log(
              `[gapp-preload] Loaded ${methods.length} RPCs for ${path}:`,
              methods
            );
          } else {
            console.log(`[gapp-preload] No RPCs matched for ${path}`);
          }

          const script = `<script>window.__PRELOADED__ = ${JSON.stringify(
            preloaded
          )};window.__PRELOAD_TIMESTAMP__ = ${Date.now()};</script>`;
          return html.replace("</head>", `${script}</head>`);
        } catch (err) {
          if ((err as Error).name === "TimeoutError") {
            console.warn("[gapp-preload] Server timeout, skipping preload");
          } else {
            console.warn(
              "[gapp-preload] Server unavailable, skipping preload:",
              (err as Error).message
            );
          }
          return html;
        }
      },
    },
  };
}
