package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	gap "github.com/germtb/gap"
	"github.com/germtb/siauth"
	pb "with-auth/server/generated"
	"google.golang.org/protobuf/proto"
)

type App struct {
	mu     sync.Mutex
	items  []*pb.Item
	nextID int
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// --- Auth setup ---
	dataRoot := os.Getenv("DATA_ROOT")
	if dataRoot == "" {
		home, _ := os.UserHomeDir()
		dataRoot = filepath.Join(home, ".with-auth-example")
	}
	os.MkdirAll(dataRoot, 0755)

	pepper := loadOrCreatePepper(dataRoot)
	auth, err := siauth.InitWithRoot(pepper, "with-auth", dataRoot)
	if err != nil {
		slog.Error("Failed to initialize auth", "error", err)
		os.Exit(1)
	}

	authServer := &siauth.AuthRpcServer{
		Auth:          auth,
		SecureCookies: false, // dev mode
	}

	app := &App{}
	dispatcher := gap.NewDispatcher()

	// Auth middleware: validate token on every request, store in context if valid.
	// Uses gap.AuthMiddleware which accepts any func(r) -> any.
	dispatcher.Use(gap.AuthMiddleware(func(r *http.Request) any {
		token, _ := siauth.ValidateAuthToken(r, auth)
		return token // nil if not authenticated — that's fine
	}))

	// Public handler — anyone can read items
	dispatcher.Unary["GetItems"] = func(w http.ResponseWriter, r *http.Request, method string, body []byte) ([]byte, error) {
		app.mu.Lock()
		defer app.mu.Unlock()
		resp := &pb.GetItemsResponse{Items: app.items}
		return proto.Marshal(resp)
	}

	// Protected handler — only authenticated users can create items.
	// gap.RequireAuth returns 401 if no token in context.
	dispatcher.Unary["CreateItem"] = gap.RequireAuth(
		func(w http.ResponseWriter, r *http.Request, method string, body []byte) ([]byte, error) {
			var req pb.CreateItemRequest
			if err := proto.Unmarshal(body, &req); err != nil {
				return nil, gap.ErrValidation("invalid request body")
			}

			// Token is guaranteed non-nil inside RequireAuth
			token := gap.GetAuthToken(r).(*siauth.Token)

			app.mu.Lock()
			defer app.mu.Unlock()
			app.nextID++
			item := &pb.Item{
				Id:        fmt.Sprintf("%d", app.nextID),
				Title:     req.Title,
				CreatedBy: token.Username,
			}
			app.items = append(app.items, item)
			resp := &pb.CreateItemResponse{Item: item}
			return proto.Marshal(resp)
		},
	)

	preload := gap.NewPreloadEngine(gap.PreloadEngineConfig{
		Routes: pb.RoutePreloads,
		PreloadFunc: func(ctx context.Context, r *http.Request, method string, params map[string]string) (proto.Message, proto.Message, error) {
			body, err := dispatcher.Unary[method](nil, r, method, nil)
			if err != nil {
				return nil, nil, err
			}
			switch method {
			case "GetItems":
				req := &pb.GetItemsRequest{}
				resp := &pb.GetItemsResponse{}
				if err := proto.Unmarshal(body, resp); err != nil {
					return nil, nil, err
				}
				return req, resp, nil
			default:
				return nil, nil, fmt.Errorf("unknown preload method: %s", method)
			}
		},
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("public/assets"))))

	// Auth RPC endpoint — siauth has its own internal RPC dispatcher
	mux.HandleFunc("/rpc/auth", authServer.HandleRpc)

	// App RPC endpoint
	mux.Handle("/rpc", dispatcher)

	mux.HandleFunc("/__preload", preload.HandlePreloadEndpoint)
	mux.HandleFunc("/", preload.ServeHTML)

	addr := ":" + port
	slog.Info("Server starting", "url", "http://localhost:"+port)
	if err := gap.ListenAndServe(addr, mux); err != http.ErrServerClosed {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func loadOrCreatePepper(dataRoot string) [32]byte {
	pepperPath := filepath.Join(dataRoot, ".pepper")
	data, err := os.ReadFile(pepperPath)
	if err == nil && len(data) == 32 {
		var pepper [32]byte
		copy(pepper[:], data)
		return pepper
	}
	var pepper [32]byte
	rand.Read(pepper[:])
	os.WriteFile(pepperPath, pepper[:], 0600)
	return pepper
}
