import { useState } from "react";
import { authRpc } from "../rpc";

export function LoginRoute() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isSignup, setIsSignup] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      if (isSignup) {
        await authRpc.Signup({ username, password });
      } else {
        await authRpc.Login({ username, password });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Authentication failed");
    }
  };

  return (
    <div style={{ padding: "2rem", maxWidth: "400px", margin: "0 auto" }}>
      <h1>{isSignup ? "Sign Up" : "Log In"}</h1>
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: "1rem" }}>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Username"
            style={{ padding: "0.5rem", width: "100%", boxSizing: "border-box" }}
          />
        </div>
        <div style={{ marginBottom: "1rem" }}>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Password"
            style={{ padding: "0.5rem", width: "100%", boxSizing: "border-box" }}
          />
        </div>
        {error && <p style={{ color: "red" }}>{error}</p>}
        <button type="submit" style={{ padding: "0.5rem 1rem", marginRight: "0.5rem" }}>
          {isSignup ? "Sign Up" : "Log In"}
        </button>
        <button type="button" onClick={() => setIsSignup(!isSignup)} style={{ padding: "0.5rem 1rem" }}>
          {isSignup ? "Have an account? Log in" : "Need an account? Sign up"}
        </button>
      </form>
    </div>
  );
}
