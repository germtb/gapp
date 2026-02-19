import { createRpcTransport, createRpcProxy } from "@gap/client";
import { AuthClientImpl } from "siauth-ts";
import { AppServiceClientImpl } from "./generated/service";

// App RPCs → /rpc
const transport = createRpcTransport({ url: "/rpc" });
const baseClient = new AppServiceClientImpl(transport);
export const rpc = createRpcProxy(baseClient);

// Auth RPCs → /rpc/auth (siauth's own dispatcher)
const authTransport = createRpcTransport({ url: "/rpc/auth" });
const baseAuthClient = new AuthClientImpl(authTransport);
export const authRpc = createRpcProxy(baseAuthClient);
