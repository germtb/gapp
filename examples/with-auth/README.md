# Gap + siauth Example

A complete, runnable gap project showing how to integrate authentication using [siauth](https://github.com/germtb/siauth).

The same pattern works with any auth library — gap provides generic interfaces
(`AuthMiddleware`, `RequireAuth`, `SetAuthToken`, `GetAuthToken`) that work with
any token type.

## Running

```bash
gap run examples/with-auth
```

## What this demonstrates

**Server (`server/main.go`):**
- `gap.AuthMiddleware(fn)` — wraps any validation function as middleware
- `gap.RequireAuth(handler)` — protects individual handlers (returns 401 if no token)
- `gap.GetAuthToken(r)` — retrieves the token inside handlers
- Mounting a separate RPC endpoint at `/rpc/auth`

**Client (`client/src/`):**
- Second `createRpcTransport` for auth RPCs (`/rpc/auth`)
- `AuthStore` reacting to auth RPC events (Status, Login, Signup, Logout)
- Auth guard pattern in app entry point
- Login/signup form component
