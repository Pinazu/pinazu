package api

import (
	"embed"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	custom_middleware "github.com/pinazu/internal/api/middleware"
	"github.com/pinazu/internal/api/websocket"
	db "github.com/pinazu/internal/db"
)

type Server struct {
	queries *db.Queries
	nc      *nats.Conn
	log     hclog.Logger
}

func NewServer(dbPool *pgxpool.Pool, nc *nats.Conn, log hclog.Logger) *Server {
	return &Server{
		queries: db.New(dbPool),
		nc:      nc,
		log:     log,
	}
}

func LoadRoutes(db *pgxpool.Pool, natsConn *nats.Conn, wsHandler *websocket.Handler, log hclog.Logger) http.Handler {
	server := NewStrictHandlerWithOptions(NewServer(db, natsConn, log), []StrictMiddlewareFunc{},
		StrictHTTPServerOptions{
			RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			},
			ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				log.Error("Response error", "error", err)
				if _, ok := err.(*NotFoundError); ok {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
			},
		},
	)

	// Create router and subrouters
	router := chi.NewRouter()
	// Use logging middleware
	router.Use(middleware.Logger)
	// Use SSE auto-flush middleware for immediate streaming
	router.Use(custom_middleware.SSEAutoFlushMiddleware())

	// Define websocket handlers
	router.Handle("/v1/ws", wsHandler)

	// Serve Swagger UI
	router.Get("/docs", redocHandler(false))
	router.Get("/docs/", redocHandler(false))
	router.Get("/docs/redoc.standalone.js", redocHandler(true))
	router.Get("/swagger/openapi.yaml", openAPISpecHandler(log))

	// Host SPA at root prefix
	router.Handle("/*", SPAHandler(log))
	return HandlerWithOptions(server, ChiServerOptions{
		BaseRouter: router,
	})
}

//go:embed swagger_doc/index.html
//go:embed swagger_doc/redoc.html
//go:embed swagger_doc/redoc.standalone.js
var content embed.FS

func redocHandler(js bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if js {
			http.ServeFileFS(w, r, content, "swagger_doc/redoc.standalone.js")
			return
		}
		http.ServeFileFS(w, r, content, "swagger_doc/redoc.html")
	}
}

func openAPISpecHandler(log hclog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("Serving openapi documents")
		w.Header().Set("Content-Type", "application/yaml")
		http.ServeFile(w, r, "api/openapi.yaml")
	}
}

func SPAHandler(log hclog.Logger) http.HandlerFunc {
	spaFS := os.DirFS("web/dist")
	return func(w http.ResponseWriter, r *http.Request) {
		// Any path not ending with a file extension is served as index.html
		if path.Ext(r.URL.Path) == "" || r.URL.Path == "/" {
			http.ServeFileFS(w, r, spaFS, "index.html")
			return
		}
		log.Info("Serving file", "path", path.Clean(r.URL.Path))
		f, err := spaFS.Open(strings.TrimPrefix(path.Clean(r.URL.Path), "/"))
		if err == nil {
			defer f.Close()
		}
		if os.IsNotExist(err) {
			w.Write([]byte("Content not found"))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.FileServer(http.FS(spaFS)).ServeHTTP(w, r)
	}
}
