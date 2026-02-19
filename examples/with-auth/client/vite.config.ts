import { defineConfig, type Plugin, type ViteDevServer, type Connect } from "vite";

let currentRequestCookies: string | undefined;

function gappPreloadPlugin(options?: {
  serverUrl?: string;
  preloadPath?: string;
}): Plugin {
  const serverUrl = options?.serverUrl ?? "http://localhost:8080";
  const preloadPath = options?.preloadPath ?? "/__preload";

  return {
    name: "gapp-preload",
    configureServer(server: ViteDevServer) {
      server.middlewares.use((req: Connect.IncomingMessage, _res, next) => {
        currentRequestCookies = req.headers.cookie;
        next();
      });
    },
    transformIndexHtml: {
      order: "pre",
      async handler(html, ctx) {
        const path = (ctx as any).originalUrl
          ? new URL((ctx as any).originalUrl, "http://localhost").pathname
          : "/";
        try {
          const url = `${serverUrl}${preloadPath}?path=${encodeURIComponent(path)}`;
          const res = await fetch(url, {
            headers: currentRequestCookies ? { Cookie: currentRequestCookies } : {},
            signal: AbortSignal.timeout(2000),
          });
          if (!res.ok) return html;
          const preloaded = await res.json();
          const script = `<script>window.__PRELOADED__ = ${JSON.stringify(preloaded)};window.__PRELOAD_TIMESTAMP__ = ${Date.now()};</script>`;
          return html.replace("</head>", `${script}</head>`);
        } catch {
          return html;
        }
      },
    },
  };
}

export default defineConfig({
  plugins: [gappPreloadPlugin()],
  server: {
    proxy: {
      "/rpc": "http://localhost:8080",
    },
  },
  build: {
    outDir: "../server/public",
    emptyOutDir: true,
  },
});
