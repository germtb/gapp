import { useState } from "react";
import { useStore } from "@gap/react";
import type { RpcDeclaration } from "@gap/client";
import { rpc, authRpc } from "../rpc";
import { itemStore } from "../stores/ItemStore";
import { authStore } from "../stores/AuthStore";

export const homeRoute = {
  path: "/",
  factory: () => ({
    component: HomeRoute,
    rpcs: [
      { method: "GetItems" },
    ] as RpcDeclaration[],
  }),
};

export function HomeRoute() {
  const { items } = useStore(itemStore);
  const { username } = useStore(authStore);
  const [title, setTitle] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) return;
    await rpc.CreateItem({ title: title.trim() });
    setTitle("");
    await rpc.GetItems({});
  };

  return (
    <div style={{ padding: "2rem", maxWidth: "600px", margin: "0 auto" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h1>Items</h1>
        <div>
          <span style={{ marginRight: "1rem" }}>{username}</span>
          <button onClick={() => authRpc.Logout({})} style={{ padding: "0.5rem 1rem" }}>
            Logout
          </button>
        </div>
      </div>
      <form onSubmit={handleSubmit} style={{ marginBottom: "1rem" }}>
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="New item..."
          style={{ padding: "0.5rem", marginRight: "0.5rem" }}
        />
        <button type="submit" style={{ padding: "0.5rem 1rem" }}>
          Add
        </button>
      </form>
      <ul>
        {items.map((item) => (
          <li key={item.id}>
            {item.title}
            {item.createdBy && <span style={{ color: "#888", marginLeft: "0.5rem" }}>— {item.createdBy}</span>}
          </li>
        ))}
      </ul>
    </div>
  );
}
