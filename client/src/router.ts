import { createCallbackSet } from "./callbackSet";

// Extract params from a single path segment
type ExtractSegmentParams<S extends string> = S extends `:${infer Param}?`
  ? { [k in Param]?: string }
  : S extends `:${infer Param}`
  ? { [k in Param]: string }
  : {};

// Extract params from route path by splitting on / and processing each segment
// Supports optional params with :param? syntax
type ExtractParams<S extends string> = S extends `${infer Head}/${infer Rest}`
  ? ExtractSegmentParams<Head> & ExtractParams<Rest>
  : ExtractSegmentParams<S>;

export type Route<Path extends string, Metadata> = {
  factory: (props: ExtractParams<Path>) => Metadata;
  path: Path;
};

export type RouteTree<Metadata> = {
  node: Route<string, Metadata> | null;
  static: {
    [key: string]: RouteTree<Metadata>;
  };
  dynamic: {
    key: string;
    optional: boolean;
    value: RouteTree<Metadata>;
  } | null;
};

type WindowAPI = {
  getPathname: () => string;
  navigate: (to: string) => void;
  addEventListener: (
    type: string,
    listener: (this: Window, ev: Event) => any
  ) => void;
  removeEventListener: (
    type: string,
    listener: (this: Window, ev: Event) => any
  ) => void;
};

const BrowserWindowAPI: WindowAPI = {
  getPathname: () => window.location.pathname,
  navigate: (to: string) => {
    window.history.pushState(null, "", to);
    window.dispatchEvent(new Event("popstate"));
  },
  addEventListener: window.addEventListener.bind(window),
  removeEventListener: window.removeEventListener.bind(window),
};

export class Router<Metadata> {
  private windowAPI: WindowAPI;
  public tree: RouteTree<Metadata>;
  private subscribers = createCallbackSet<Metadata>();
  private cache: Set<Route<string, Metadata>> = new Set();
  private metadataCache: Map<string, Metadata> = new Map();

  constructor(
    routes: Route<string, Metadata>[],
    windowAPI: WindowAPI = BrowserWindowAPI
  ) {
    this.windowAPI = windowAPI;

    if (routes.length === 0) {
      throw new Error(
        "At least one route is required to build the route tree."
      );
    }

    const firstRoute = routes[0]!;
    if (firstRoute.path !== "/") {
      throw new Error("The first route must be the root route with path '/'");
    }

    this.tree = {
      node: firstRoute,
      static: {},
      dynamic: null,
    };

    for (const route of routes.slice(1)) {
      this.addRouteToTree(route);
    }

    for (const route of routes) {
      this.cache.add(route);
    }

    this.windowAPI.addEventListener("popstate", this.onPopstate);
  }

  cleanup() {
    this.windowAPI.removeEventListener("popstate", this.onPopstate);
    this.subscribers.clear();
    this.cache.clear();
    this.metadataCache.clear();
    this.tree = { node: null, static: {}, dynamic: null };
  }

  private onPopstate = () => {
    this.subscribers.call(this.current());
  };

  url<Path extends string>(
    route: Route<Path, any>,
    params: ExtractParams<Path>,
    queryParams?: { [key: string]: string | number | boolean }
  ): string {
    if (!this.cache.has(route)) {
      throw new Error("Route not registered in the router");
    }

    const parts = route.path.split("/").filter(Boolean);
    let urlParts: string[] = [];

    for (const part of parts) {
      if (part.startsWith(":")) {
        const isOptional = part.endsWith("?");
        const paramName = isOptional ? part.slice(1, -1) : part.slice(1);
        const paramValue = params[paramName as keyof typeof params];

        if (!paramValue) {
          if (isOptional) {
            // Stop building URL when we hit an unprovided optional param
            break;
          }
          throw new Error(
            `Missing required parameter '${paramName}' for route '${route.path}'`
          );
        }
        urlParts.push(paramValue as string);
      } else {
        urlParts.push(part);
      }
    }

    const baseUrl = "/" + urlParts.join("/");

    if (queryParams && Object.keys(queryParams).length > 0) {
      const queryString = Object.entries(queryParams)
        .map(
          ([key, value]) =>
            `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`
        )
        .join("&");
      return `${baseUrl}?${queryString}`;
    }

    return baseUrl;
  }

  onNavigate(callback: (metadata: Metadata) => void): () => void {
    const unsubscribe = this.subscribers.add(callback);
    callback(this.current());
    return unsubscribe;
  }

  current(): Metadata {
    const pathname = this.windowAPI.getPathname();
    const result = this.find(pathname);
    if (!result) {
      throw new Error(`No route found for pathname: ${pathname}`);
    }
    return result;
  }

  path(): string {
    const pathname = this.windowAPI.getPathname();
    return pathname;
  }

  navigate(path: string) {
    this.windowAPI.navigate(path);
    this.subscribers.call(this.current());
  }

  find(path: string): Metadata | null {
    const cached = this.metadataCache.get(path);
    if (cached) {
      return cached;
    }

    if (path === "/") {
      const metadata = this.tree.node!.factory({});
      this.metadataCache.set(path, metadata);
      return metadata;
    }

    const segments = path.split("/");

    let current: RouteTree<Metadata> = {
      node: null,
      static: {
        "": this.tree,
      },
      dynamic: null,
    };

    for (const segment of segments) {
      const child = current.static[segment];
      if (child) {
        current = child;
      } else if (current.dynamic) {
        current = current.dynamic.value;
      } else {
        return null;
      }
    }

    if (current.node == null) {
      return null;
    }

    const params = this.findParams(current.node.path, path);
    const metadata = current.node.factory(params);
    this.metadataCache.set(path, metadata);
    return metadata;
  }

  private findParams(
    routePath: string,
    pathname: string
  ): { [key: string]: string | undefined } {
    const params: { [key: string]: string | undefined } = {};
    const routePathSegments = routePath.split("/").filter(Boolean);
    const pathnameSegments = pathname.split("/").filter(Boolean);

    for (let i = 0; i < routePathSegments.length; i++) {
      const routeSegment = routePathSegments[i]!;
      const pathSegment = pathnameSegments[i];

      if (routeSegment.startsWith(":")) {
        // Remove the optional marker (?) if present
        const isOptional = routeSegment.endsWith("?");
        const paramName = isOptional
          ? routeSegment.slice(1, -1)
          : routeSegment.slice(1);

        // If we have a path segment, use it; otherwise it's undefined (optional param not provided)
        params[paramName] = pathSegment;
      }
    }

    return params;
  }

  private addRouteToTree(route: Route<string, Metadata>) {
    const parts = route.path.split("/").filter(Boolean);
    let current: RouteTree<Metadata> = this.tree;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i]!;
      const isOptional = part.endsWith("?");
      const isDynamic = part.startsWith(":");

      if (isDynamic) {
        // Normalize the key (remove the ? suffix for comparison)
        const key = isOptional ? part.slice(0, -1) : part;

        if (current.dynamic) {
          const existingKey = current.dynamic.key;
          if (existingKey !== key) {
            throw new Error(
              `Conflicting dynamic segments: '${existingKey}' and '${key}'`
            );
          }
        } else {
          current.dynamic = {
            key,
            optional: isOptional,
            value: { node: null, static: {}, dynamic: null },
          };
        }

        // If this param is optional, the current node is also a valid endpoint
        if (isOptional && !current.node) {
          current.node = route;
        }

        current = current.dynamic.value;
      } else {
        if (!current.static[part]) {
          current.static[part] = {
            node: null,
            static: {},
            dynamic: null,
          };
        }
        current = current.static[part]!;
      }
    }

    if (current.node && current.node !== route) {
      throw new Error(`Conflicting routes for path: '${route.path}'`);
    }

    current.node = route;
  }
}
