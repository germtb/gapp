#!/usr/bin/env bun
/**
 * Generates a Go file with route preload configuration from TypeScript route files.
 *
 * Scans route files for `rpcs` arrays in route definitions and outputs a Go file
 * with the route-to-RPC mapping for server-side preloading.
 *
 * Usage:
 *   bun run generate-preload-config.ts --routes-dir <path> --output <path> [--package <name>]
 *
 * Example:
 *   bun run generate-preload-config.ts --routes-dir ./src/routes --output ../server/generated/preload_routes.go
 */

import ts from "typescript";
import { readFileSync, writeFileSync, readdirSync } from "fs";
import { join } from "path";

interface RpcSpec {
  method: string;
  params?: Record<string, string>;
}

interface RoutePreload {
  path: string;
  rpcs: RpcSpec[];
}

// Parse CLI args
function parseArgs(): { routesDir: string; output: string; packageName: string } {
  const args = process.argv.slice(2);
  let routesDir = "";
  let output = "";
  let packageName = "generated";

  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--routes-dir" && i + 1 < args.length) {
      routesDir = args[++i];
    } else if (args[i] === "--output" && i + 1 < args.length) {
      output = args[++i];
    } else if (args[i] === "--package" && i + 1 < args.length) {
      packageName = args[++i];
    }
  }

  if (!routesDir || !output) {
    console.error("Usage: generate-preload-config.ts --routes-dir <path> --output <path> [--package <name>]");
    process.exit(1);
  }

  return { routesDir, output, packageName };
}

function extractRpcsFromObject(obj: ts.ObjectLiteralExpression, sourceFile: ts.SourceFile): RpcSpec[] {
  const rpcs: RpcSpec[] = [];

  for (const prop of obj.properties) {
    if (!ts.isPropertyAssignment(prop)) continue;

    const propName = prop.name.getText(sourceFile);

    if (propName === "rpcs") {
      let arrayExpr: ts.ArrayLiteralExpression | null = null;
      if (ts.isArrayLiteralExpression(prop.initializer)) {
        arrayExpr = prop.initializer;
      } else if (ts.isAsExpression(prop.initializer) && ts.isArrayLiteralExpression(prop.initializer.expression)) {
        arrayExpr = prop.initializer.expression;
      }
      if (!arrayExpr) continue;

      for (const element of arrayExpr.elements) {
        if (ts.isObjectLiteralExpression(element)) {
          const rpc: RpcSpec = { method: "" };

          for (const rpcProp of element.properties) {
            if (!ts.isPropertyAssignment(rpcProp)) continue;

            const rpcPropName = rpcProp.name.getText(sourceFile);

            if (rpcPropName === "method" && ts.isStringLiteral(rpcProp.initializer)) {
              rpc.method = rpcProp.initializer.text;
            }

            if (rpcPropName === "params" && ts.isObjectLiteralExpression(rpcProp.initializer)) {
              rpc.params = {};
              for (const paramProp of rpcProp.initializer.properties) {
                if (ts.isPropertyAssignment(paramProp) && ts.isStringLiteral(paramProp.initializer)) {
                  const paramName = paramProp.name.getText(sourceFile);
                  rpc.params[paramName] = paramProp.initializer.text;
                }
              }
            }
          }

          if (rpc.method) {
            rpcs.push(rpc);
          }
        }
      }
    }
  }

  return rpcs;
}

function parseRouteFile(filePath: string): RoutePreload | null {
  const content = readFileSync(filePath, "utf-8");
  const sourceFile = ts.createSourceFile(
    filePath,
    content,
    ts.ScriptTarget.Latest,
    true
  );

  let path: string | null = null;
  const rpcs: RpcSpec[] = [];

  function visit(node: ts.Node) {
    if (ts.isVariableDeclaration(node) && node.initializer) {
      if (ts.isObjectLiteralExpression(node.initializer)) {
        const obj = node.initializer;

        for (const prop of obj.properties) {
          if (!ts.isPropertyAssignment(prop)) continue;

          const propName = prop.name.getText(sourceFile);

          if (propName === "path" && ts.isStringLiteral(prop.initializer)) {
            path = prop.initializer.text;
          }

          if (propName === "factory" && ts.isArrowFunction(prop.initializer)) {
            const arrow = prop.initializer;

            if (ts.isParenthesizedExpression(arrow.body)) {
              const inner = arrow.body.expression;
              if (ts.isObjectLiteralExpression(inner)) {
                rpcs.push(...extractRpcsFromObject(inner, sourceFile));
              }
            } else if (ts.isObjectLiteralExpression(arrow.body)) {
              rpcs.push(...extractRpcsFromObject(arrow.body, sourceFile));
            } else if (ts.isBlock(arrow.body)) {
              for (const stmt of arrow.body.statements) {
                if (ts.isReturnStatement(stmt) && stmt.expression && ts.isObjectLiteralExpression(stmt.expression)) {
                  rpcs.push(...extractRpcsFromObject(stmt.expression, sourceFile));
                }
              }
            }
          }
        }
      }
    }

    ts.forEachChild(node, visit);
  }

  visit(sourceFile);

  if (!path) {
    return null;
  }

  return { path, rpcs };
}

function generateGoCode(routes: RoutePreload[], packageName: string): string {
  const allMethods = new Set<string>();
  for (const route of routes) {
    for (const rpc of route.rpcs) {
      allMethods.add(rpc.method);
    }
  }
  const sortedMethods = Array.from(allMethods).sort();

  let code = `// Code generated by generate-preload-config.ts. DO NOT EDIT.

package ${packageName}

// RpcSpec defines an RPC to preload with optional parameter mappings
type RpcSpec struct {
	Method string
	Params map[string]string
}

// RoutePreload defines preload configuration for a route pattern
type RoutePreload struct {
	Pattern string
	Rpcs    []RpcSpec
}

// RoutePreloads contains all route preload configurations
var RoutePreloads = []RoutePreload{
`;

  for (const route of routes) {
    if (route.rpcs.length === 0) continue;

    code += `	{
		Pattern: "${route.path}",
		Rpcs: []RpcSpec{
`;
    for (const rpc of route.rpcs) {
      const params = rpc.params
        ? `map[string]string{${Object.entries(rpc.params)
            .map(([k, v]) => `"${k}": "${v}"`)
            .join(", ")}}`
        : "nil";
      code += `			{Method: "${rpc.method}", Params: ${params}},
`;
    }
    code += `		},
	},
`;
  }

  code += `}

// PreloadMethods contains all unique RPC methods that need preload handlers
var PreloadMethods = []string{
`;

  for (const method of sortedMethods) {
    code += `	"${method}",
`;
  }

  code += `}
`;

  return code;
}

// Main
const { routesDir, output, packageName } = parseArgs();

console.log("Generating preload configuration...");
console.log(`  Routes dir: ${routesDir}`);
console.log(`  Output: ${output}`);

const routeFiles = readdirSync(routesDir).filter((f) => f.endsWith(".tsx") || f.endsWith(".ts"));
const routes: RoutePreload[] = [];

for (const file of routeFiles) {
  const filePath = join(routesDir, file);
  const route = parseRouteFile(filePath);
  if (route && route.rpcs.length > 0) {
    routes.push(route);
    console.log(`  ${file}: ${route.rpcs.length} RPCs`);
  }
}

if (routes.length === 0) {
  console.log("  No routes with RPC declarations found.");
  process.exit(0);
}

const goCode = generateGoCode(routes, packageName);
writeFileSync(output, goCode);

console.log("");
console.log(`Generated ${output}`);
console.log(`  ${routes.length} routes, ${new Set(routes.flatMap((r) => r.rpcs.map((rpc) => rpc.method))).size} unique RPC methods`);
