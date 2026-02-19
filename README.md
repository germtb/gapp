# gap

Full-stack Go + TypeScript framework with protobuf RPC, code generation, and preloading.

## Features

- **Type-safe RPCs** — Define services in protobuf, get generated Go handlers and TypeScript clients
- **Code generation** — Single `gap codegen` command generates Go and TypeScript from `.proto` files
- **Preloading** — Server-side data preloading with route-aware RPC batching
- **React hooks** — `useStore` bindings that auto-update on RPC responses
- **Client-side routing** — Type-safe router with parameter extraction
- **Vite plugin** — Dev-mode preload injection via `@gap/client/vite`

## Quick Start

```bash
# Install the CLI
go install github.com/germtb/gap/cli@latest

# Create a new project
gap init myapp --framework react -y

# Start development
cd myapp
gap run
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
2. Run `gap codegen` to generate Go structs and TypeScript types
3. Implement RPC handlers in `server/main.go`
4. Use generated types in your client code

## Packages

| Package | Description |
|---------|-------------|
| `github.com/germtb/gap` | Go server framework — dispatcher, preload engine, auth middleware |
| `@gap/client` | Client runtime — stores, RPC transport, router, preloading |
| `@gap/react` | React bindings — `useStore` hook |
| `@gap/codegen` | Code generation utilities |

## CLI Commands

| Command | Description |
|---------|-------------|
| `gap init <name>` | Create a new project (react or vanilla) |
| `gap codegen` | Generate Go + TypeScript from protobuf |
| `gap run [path]` | Start server and client dev server |
| `gap build [path]` | Build for production |

## Examples

See the [`examples/`](./examples) directory:

- **with-auth** — Authentication with siauth, protected RPC handlers

## License

MIT
