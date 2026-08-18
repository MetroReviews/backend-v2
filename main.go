package main

import (
	"net/http"
	"os"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/bot"
	"github.com/MetroReviews/backend-v2/routes/actions"
	"github.com/MetroReviews/backend-v2/routes/bots"
	"github.com/MetroReviews/backend-v2/routes/businesses"
	"github.com/MetroReviews/backend-v2/routes/categories"
	"github.com/MetroReviews/backend-v2/routes/panel"
	"github.com/MetroReviews/backend-v2/routes/projects"
	"github.com/MetroReviews/backend-v2/routes/reviews"
	rolesroutes "github.com/MetroReviews/backend-v2/routes/roles"
	"github.com/MetroReviews/backend-v2/routes/team"
	webhookroutes "github.com/MetroReviews/backend-v2/routes/webhooks"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/infinitybotlist/eureka/zapchi"
	"go.uber.org/zap"
)

var routers = []uapi.APIRouter{
	businesses.Router{},
	bots.Router{},
	reviews.Router{},
	categories.Router{},
	actions.Router{},
	team.Router{},
	panel.Router{},
	rolesroutes.Router{},
	projects.Router{},
	webhookroutes.Router{},
}

func main() {
	secretsPath := os.Getenv("SECRETS_FILE")
	if secretsPath == "" {
		secretsPath = "secrets.json"
	}
	if len(os.Args) > 1 {
		secretsPath = os.Args[1]
	}

	if err := state.Setup(); err != nil {
		panic("failed to set up state: " + err.Error())
	}
	defer state.Pool.Close()

	if state.Config.Discord.Token != "" {
		if err := bot.Setup(); err != nil {
			state.Logger.Fatal("failed to set up bot", zap.Error(err))
		}
		if err := bot.Open(); err != nil {
			state.Logger.Fatal("failed to open Discord connection", zap.Error(err))
		}
		defer bot.Close()
	} else {
		state.Logger.Warn("no bot token configured; running API only (bot-dependent endpoints will fail)")
	}

	api.Setup()

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(zapchi.Logger(state.Logger, "api"))
	r.Use(corsMiddleware)

	for _, router := range routers {
		name, desc := router.Tag()
		docs.AddTag(name, desc)
		uapi.State.SetCurrentTag(name)
		router.Routes(r)
	}

	r.Get("/openapi", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := jsonimpl.MarshalToWriter(w, docs.GetSchema()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// Human-facing rendering of /openapi. Redoc is loaded from its CDN by
	// the browser that opens this page, not by the server — this handler
	// itself stays a static, dependency-free HTML string.
	r.Get("/docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(docsHTML)) //nolint:errcheck
	})

	state.Logger.Info("listening", zap.String("addr", state.Config.Server.BindAddr))
	if err := http.ListenAndServe(state.Config.Server.BindAddr, r); err != nil {
		state.Logger.Fatal("server exited", zap.Error(err))
	}
}

// docsHTML renders /openapi with Redoc — a single <redoc> tag pointed at
// the JSON endpoint above, plus its CDN-hosted bundle. No build step and
// no assets to embed: the browser fetches both the schema and the
// renderer itself at load time.
const docsHTML = `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>Metro Reviews API Docs</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>body { margin: 0; padding: 0; }</style>
</head>
<body>
	<redoc spec-url="/openapi"></redoc>
	<script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>
`

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
