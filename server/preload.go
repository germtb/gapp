package gap

import (
	"bytes"
	"compress/gzip"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

// ViteManifest represents the Vite build manifest
type ViteManifest map[string]ViteManifestEntry

type ViteManifestEntry struct {
	File string   `json:"file"`
	Src  string   `json:"src"`
	CSS  []string `json:"css"`
}

// Assets holds the resolved asset paths from Vite manifest
type Assets struct {
	JS  string
	CSS string
}

//go:embed template.html
var templateFS embed.FS

// PreloadedRpc contains base64-encoded gzip-compressed protobuf bytes for request and response
type PreloadedRpc struct {
	RequestBytes  string `json:"requestBytes"`
	ResponseBytes string `json:"responseBytes"`
}

// ToProtoBytes marshals a proto message, gzip-compresses it, and base64-encodes the result.
func ToProtoBytes(v any) string {
	if v == nil {
		return ""
	}
	if msg, ok := v.(proto.Message); ok {
		protoBytes, err := proto.Marshal(msg)
		if err != nil {
			slog.Error("Failed to marshal proto message", "error", err)
			return ""
		}

		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(protoBytes); err != nil {
			slog.Error("Failed to gzip compress", "error", err)
			return ""
		}
		if err := gz.Close(); err != nil {
			slog.Error("Failed to close gzip writer", "error", err)
			return ""
		}

		return base64.StdEncoding.EncodeToString(buf.Bytes())
	}
	slog.Error("ToProtoBytes called with non-proto value")
	return ""
}

// RouteSpec defines preload configuration for a route pattern.
type RouteSpec struct {
	Pattern string
	Rpcs    []RpcSpec
}

// RpcSpec defines an RPC to preload with optional parameter mappings.
type RpcSpec struct {
	Method string
	Params map[string]string
}

// PreloadFunc is the callback that executes an RPC for preloading.
// It receives the context, method name, and substituted route params.
// It returns the request and response proto messages.
type PreloadFunc func(ctx context.Context, r *http.Request, method string, params map[string]string) (request, response proto.Message, err error)

// PreloadEngine handles route-based RPC preloading and HTML rendering.
type PreloadEngine struct {
	Routes      []RouteSpec
	PreloadFunc PreloadFunc
	tmpl        *template.Template
	assets      Assets
}

type PreloadEngineConfig struct {
	Routes       []RouteSpec
	PreloadFunc  PreloadFunc
	ManifestPath string // path to .vite/manifest.json, defaults to "public/.vite/manifest.json"
	AppName      string // defaults to "App"
}

func NewPreloadEngine(config PreloadEngineConfig) *PreloadEngine {
	tmpl := template.Must(template.ParseFS(templateFS, "template.html"))
	manifestPath := config.ManifestPath
	if manifestPath == "" {
		manifestPath = "public/.vite/manifest.json"
	}
	assets := LoadAssetsFromManifest(manifestPath)
	return &PreloadEngine{
		Routes:      config.Routes,
		PreloadFunc: config.PreloadFunc,
		tmpl:        tmpl,
		assets:      assets,
	}
}

// LoadAssetsFromManifest reads the Vite manifest to get hashed asset filenames.
func LoadAssetsFromManifest(manifestPath string) Assets {
	assets := Assets{
		JS:  "/assets/index.js",
		CSS: "/assets/index.css",
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		slog.Info("Vite manifest not found, using default assets", "error", err)
		return assets
	}

	var manifest ViteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		slog.Error("Failed to parse Vite manifest", "error", err)
		return assets
	}

	if entry, ok := manifest["index.html"]; ok {
		assets.JS = "/" + entry.File
		if len(entry.CSS) > 0 {
			assets.CSS = "/" + entry.CSS[0]
		}
		slog.Info("Loaded assets from Vite manifest", "js", assets.JS, "css", assets.CSS)
	}

	return assets
}

// ServeHTML serves the HTML page with preloaded data for the matched route.
func (p *PreloadEngine) ServeHTML(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/assets/") ||
		strings.HasPrefix(r.URL.Path, "/rpc") ||
		strings.HasPrefix(r.URL.Path, "/__preload") {
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	preloaded := p.executeForPath(ctx, r)
	p.renderHTML(w, preloaded)
}

// HandlePreloadEndpoint handles the /__preload?path=... endpoint used by the Vite plugin in dev mode.
func (p *PreloadEngine) HandlePreloadEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	fakeReq := r.Clone(ctx)
	fakeReq.URL.Path = path

	preloaded := p.executeForPath(ctx, fakeReq)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	json.NewEncoder(w).Encode(preloaded)
}

func (p *PreloadEngine) executeForPath(ctx context.Context, r *http.Request) map[string]PreloadedRpc {
	preloaded := make(map[string]PreloadedRpc)
	var mu sync.Mutex
	var wg sync.WaitGroup

	route, routeParams := MatchRoute(p.Routes, r.URL.Path)
	if route == nil {
		return preloaded
	}

	for _, rpcSpec := range route.Rpcs {
		rpcSpec := rpcSpec

		wg.Add(1)
		go func() {
			defer wg.Done()

			rpcParams := SubstituteParams(rpcSpec.Params, routeParams)

			if HasUnsubstitutedParam(rpcParams) {
				slog.Info("Preload: Skipping - unsubstituted params", "method", rpcSpec.Method, "params", rpcParams)
				return
			}

			req, resp, err := p.PreloadFunc(ctx, r, rpcSpec.Method, rpcParams)
			if err != nil {
				slog.Info("Preload: Failed", "method", rpcSpec.Method, "error", err)
				return
			}

			mu.Lock()
			preloaded[rpcSpec.Method] = PreloadedRpc{
				RequestBytes:  ToProtoBytes(req),
				ResponseBytes: ToProtoBytes(resp),
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return preloaded
}

func (p *PreloadEngine) renderHTML(w http.ResponseWriter, preloaded map[string]PreloadedRpc) {
	jsonBytes, _ := json.Marshal(preloaded)

	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "App"
	}

	data := struct {
		PreloadedJSON template.JS
		Timestamp     int64
		AssetsJS      string
		AssetsCSS     string
		AppName       string
	}{
		PreloadedJSON: template.JS(jsonBytes),
		Timestamp:     time.Now().UnixMilli(),
		AssetsJS:      p.assets.JS,
		AssetsCSS:     p.assets.CSS,
		AppName:       appName,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	if err := p.tmpl.Execute(w, data); err != nil {
		slog.Error("Failed to render HTML template", "error", err)
		http.Error(w, "Internal Server Error", 500)
	}
}

// MatchRoute finds the first matching route for a given path.
func MatchRoute(routes []RouteSpec, path string) (*RouteSpec, map[string]string) {
	for i := range routes {
		route := &routes[i]
		if params, ok := MatchPattern(route.Pattern, path); ok {
			return route, params
		}
	}
	return nil, nil
}

// MatchPattern matches a URL pattern against a path, extracting route parameters.
func MatchPattern(pattern, path string) (map[string]string, bool) {
	params := make(map[string]string)

	patternParts := SplitPath(pattern)
	pathParts := SplitPath(path)

	pi := 0
	for _, pp := range patternParts {
		if strings.HasPrefix(pp, ":") {
			paramName := strings.TrimSuffix(strings.TrimPrefix(pp, ":"), "?")
			optional := strings.HasSuffix(pp, "?")

			if pi < len(pathParts) {
				params[paramName] = pathParts[pi]
				pi++
			} else if !optional {
				return nil, false
			}
		} else {
			if pi >= len(pathParts) || pathParts[pi] != pp {
				return nil, false
			}
			pi++
		}
	}

	if pi != len(pathParts) {
		return nil, false
	}

	return params, true
}

// SplitPath splits a URL path into segments, trimming leading/trailing slashes.
func SplitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

// SubstituteParams replaces :param placeholders in RPC params with actual route parameter values.
func SubstituteParams(rpcParams map[string]string, routeParams map[string]string) map[string]string {
	if rpcParams == nil {
		return nil
	}
	result := make(map[string]string)
	for key, value := range rpcParams {
		for paramName, paramValue := range routeParams {
			value = strings.ReplaceAll(value, ":"+paramName, paramValue)
		}
		result[key] = value
	}
	return result
}

// HasUnsubstitutedParam checks if any parameter values still contain unresolved :param placeholders.
func HasUnsubstitutedParam(params map[string]string) bool {
	for _, v := range params {
		if strings.Contains(v, ":") {
			return true
		}
	}
	return false
}
