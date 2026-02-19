# gapp

Full-stack Go + TypeScript framework with protobuf RPC, code generation, and preloading.

## Features

- **Type-safe RPCs** — Define services in protobuf, get generated Go handlers and TypeScript clients
- **Code generation** — Single `gapp codegen` command generates Go and TypeScript from `.proto` files
- **Preloading** — Server-side data preloading with route-aware RPC batching
- **React hooks** — `useStore` bindings that auto-update on RPC responses
- **Client-side routing** — Type-safe router with parameter extraction
- **Vite plugin** — Dev-mode preload injection via `@gapp/client/vite`

## Quick Start

```bash
# Install the CLI
go install github.com/germtb/gapp/cli@latest

# Create a new project
gapp init myapp --framework react -y

# Start development
cd myapp
gapp run
```

## Architecture

```
myapp/
├── proto/service.proto     # Service definitions
├── server/
│   ├── main.go             # Go server with RPC handlers
│   └── generated/          # Generated protobuf code
└── client/
    ├── src/
    │   ├── generated/      # Generated TypeScript types
    │   ├── routes/         # Route definitions with RPC declarations
    │   └── stores/         # Reactive stores
    └── vite.config.ts
```

1. Define your service in `proto/service.proto`
2. Run `gapp codegen` to generate Go structs and TypeScript types
3. Implement RPC handlers in `server/main.go`
4. Use generated types in your client code

## Packages

| Package | Description |
|---------|-------------|
| `github.com/germtb/gapp` | Go server framework — dispatcher, preload engine, auth middleware |
| `@gapp/client` | Client runtime — stores, RPC transport, router, preloading |
| `@gapp/react` | React bindings — `useStore` hook |

## CLI Commands

| Command | Description |
|---------|-------------|
| `gapp init <name>` | Create a new project (react or vanilla) |
| `gapp codegen` | Generate Go + TypeScript from protobuf |
| `gapp run [path]` | Start server and client dev server |
| `gapp build [path]` | Build for production |

## Examples

See the [`examples/`](./examples) directory:

- **with-auth** — Authentication with siauth, protected RPC handlers

## License

MIT
