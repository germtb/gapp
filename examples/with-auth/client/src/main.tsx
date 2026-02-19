import { createRoot } from "react-dom/client";
import { dispatchPreloaded, Router, type Route } from "@gapp/client";
import { useCurrentRoute, useStore } from "@gapp/react";
import { decodePreloaded } from "./preload";
import { authRpc } from "./rpc";
import { authStore } from "./stores/AuthStore";
import { HomeRoute } from "./routes/HomeRoute";
import { LoginRoute } from "./routes/LoginRoute";
import "./stores/ItemStore";

type RouteMetadata = {
  component: () => React.ReactNode;
};

const routes: Route<string, RouteMetadata>[] = [
  {
    path: "/",
    factory: () => ({ component: HomeRoute }),
  },
];

const router = new Router(routes);

function App() {
  const { status } = useStore(authStore);
  const metadata = useCurrentRoute(router);

  if (status === "loading") return null;
  if (status === "logged-out") return <LoginRoute />;
  if (!metadata) return null;

  const Component = metadata.component;
  return <Component />;
}

async function main() {
  await dispatchPreloaded(decodePreloaded);
  authRpc.Status({});

  const root = document.getElementById("root");
  if (root) {
    createRoot(root).render(<App />);
  }
}

main();
