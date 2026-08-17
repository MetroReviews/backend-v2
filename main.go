package main

import (
	"net/http"
	"os"

	"github.com/MetroReviews/backend-v2/api"
	"github.com/MetroReviews/backend-v2/bot"
	"github.com/MetroReviews/backend-v2/routes/actions"
	"github.com/MetroReviews/backend-v2/routes/bots"
	"github.com/MetroReviews/backend-v2/routes/lists"
	"github.com/MetroReviews/backend-v2/routes/panel"
	"github.com/MetroReviews/backend-v2/routes/team"
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
	lists.Router{},
	bots.Router{},
	actions.Router{},
	team.Router{},
	panel.Router{},
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

	if state.Config.Token != "" {
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

	state.Logger.Info("listening", zap.String("addr", state.Config.BindAddr))
	if err := http.ListenAndServe(state.Config.BindAddr, r); err != nil {
		state.Logger.Fatal("server exited", zap.Error(err))
	}
}

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
