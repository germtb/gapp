import { Store } from "@gapp/client";
import { authRpc } from "../rpc";

type AuthState = {
  status: "loading" | "logged-in" | "logged-out";
  username: string | null;
};

class AuthStore extends Store<AuthState> {
  constructor() {
    super({ status: "loading", username: null });
  }

  reduceRpc(state: AuthState, event: any): AuthState {
    if (event.method === "Status" && event.result.isOk()) {
      const { isAuthenticated, username } = event.result.unwrap();
      return {
        status: isAuthenticated ? "logged-in" : "logged-out",
        username: isAuthenticated ? username : null,
      };
    }
    if (event.method === "Login" && event.result.isOk()) {
      if (event.result.unwrap().success) {
        authRpc.Status({});
      }
    }
    if (event.method === "Signup" && event.result.isOk()) {
      if (event.result.unwrap().success) {
        authRpc.Status({});
      }
    }
    if (event.method === "Logout") {
      return { status: "logged-out", username: null };
    }
    return state;
  }
}

export const authStore = new AuthStore();
