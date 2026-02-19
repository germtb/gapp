package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRouteFile(t *testing.T) {
	dir := t.TempDir()

	// React-style route
	reactRoute := `import { useState } from "react";
import { useStore } from "@gap/react";
import type { RpcDeclaration } from "@gap/client";

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
  return <div>Hello</div>;
}
`
	err := os.WriteFile(filepath.Join(dir, "HomeRoute.tsx"), []byte(reactRoute), 0644)
	if err != nil {
		t.Fatal(err)
	}

	route, err := ParseRouteFile(filepath.Join(dir, "HomeRoute.tsx"))
	if err != nil {
		t.Fatalf("ParseRouteFile failed: %v", err)
	}
	if route == nil {
		t.Fatal("Expected route, got nil")
	}
	if route.Path != "/" {
		t.Errorf("Path = %q, want %q", route.Path, "/")
	}
	if len(route.Rpcs) != 1 {
		t.Fatalf("len(Rpcs) = %d, want 1", len(route.Rpcs))
	}
	if route.Rpcs[0].Method != "GetItems" {
		t.Errorf("Method = %q, want %q", route.Rpcs[0].Method, "GetItems")
	}
}

func TestParseRouteFileMultipleRpcs(t *testing.T) {
	dir := t.TempDir()

	route := `export const userRoute = {
  path: "/users/:id",
  factory: () => ({
    component: UserPage,
    rpcs: [
      { method: "GetUser", params: { "id": "userId" } },
      { method: "GetUserPosts" },
    ] as RpcDeclaration[],
  }),
};
`
	err := os.WriteFile(filepath.Join(dir, "UserRoute.tsx"), []byte(route), 0644)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ParseRouteFile(filepath.Join(dir, "UserRoute.tsx"))
	if err != nil {
		t.Fatalf("ParseRouteFile failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected route, got nil")
	}
	if result.Path != "/users/:id" {
		t.Errorf("Path = %q, want %q", result.Path, "/users/:id")
	}
	if len(result.Rpcs) != 2 {
		t.Fatalf("len(Rpcs) = %d, want 2", len(result.Rpcs))
	}
	if result.Rpcs[0].Method != "GetUser" {
		t.Errorf("Rpcs[0].Method = %q, want %q", result.Rpcs[0].Method, "GetUser")
	}
	if result.Rpcs[0].Params["id"] != "userId" {
		t.Errorf("Rpcs[0].Params[id] = %q, want %q", result.Rpcs[0].Params["id"], "userId")
	}
	if result.Rpcs[1].Method != "GetUserPosts" {
		t.Errorf("Rpcs[1].Method = %q, want %q", result.Rpcs[1].Method, "GetUserPosts")
	}
}

func TestParseRouteFileNoRpcs(t *testing.T) {
	dir := t.TempDir()

	route := `export function NotARoute() {
  return <div>Hello</div>;
}
`
	os.WriteFile(filepath.Join(dir, "NotARoute.tsx"), []byte(route), 0644)

	result, err := ParseRouteFile(filepath.Join(dir, "NotARoute.tsx"))
	if err != nil {
		t.Fatalf("ParseRouteFile failed: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for file without route, got %+v", result)
	}
}

func TestScanRoutes(t *testing.T) {
	dir := t.TempDir()

	home := `export const homeRoute = {
  path: "/",
  factory: () => ({
    rpcs: [{ method: "GetItems" }] as RpcDeclaration[],
  }),
};
`
	about := `export const aboutRoute = {
  path: "/about",
  factory: () => ({
    rpcs: [{ method: "GetAbout" }] as RpcDeclaration[],
  }),
};
`
	// Non-route file
	utils := `export function formatDate(d: Date) { return d.toString(); }`

	os.WriteFile(filepath.Join(dir, "HomeRoute.tsx"), []byte(home), 0644)
	os.WriteFile(filepath.Join(dir, "AboutRoute.ts"), []byte(about), 0644)
	os.WriteFile(filepath.Join(dir, "utils.ts"), []byte(utils), 0644)

	routes, err := ScanRoutes(dir)
	if err != nil {
		t.Fatalf("ScanRoutes failed: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2", len(routes))
	}
}

func TestGeneratePreloadGo(t *testing.T) {
	routes := []RoutePreload{
		{
			Path: "/",
			Rpcs: []RpcSpec{
				{Method: "GetItems"},
			},
		},
		{
			Path: "/users/:id",
			Rpcs: []RpcSpec{
				{Method: "GetUser", Params: map[string]string{"id": "userId"}},
				{Method: "GetUserPosts"},
			},
		},
	}

	code := GeneratePreloadGo(routes, "generated")

	if !strings.Contains(code, "package generated") {
		t.Error("Should contain package declaration")
	}
	if !strings.Contains(code, `import gap "github.com/germtb/gap"`) {
		t.Error("Should import gap package")
	}
	if !strings.Contains(code, `[]gap.RouteSpec{`) {
		t.Error("Should use gap.RouteSpec type")
	}
	if !strings.Contains(code, `[]gap.RpcSpec{`) {
		t.Error("Should use gap.RpcSpec type")
	}
	if !strings.Contains(code, `Pattern: "/"`) {
		t.Error("Should contain root route pattern")
	}
	if !strings.Contains(code, `Method: "GetItems"`) {
		t.Error("Should contain GetItems method")
	}
	if !strings.Contains(code, `Method: "GetUser"`) {
		t.Error("Should contain GetUser method")
	}
	if !strings.Contains(code, `"id": "userId"`) {
		t.Error("Should contain params mapping")
	}
	if !strings.Contains(code, `"GetItems"`) {
		t.Error("PreloadMethods should contain GetItems")
	}
	if !strings.Contains(code, `"GetUser"`) {
		t.Error("PreloadMethods should contain GetUser")
	}
	if !strings.Contains(code, `"GetUserPosts"`) {
		t.Error("PreloadMethods should contain GetUserPosts")
	}
}
