package state

import (
	"context"
	"os"
	"time"

	"github.com/MetroReviews/backend-v2/config"
	"github.com/MetroReviews/backend-v2/migrations"
	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/validator/v10"
	"github.com/infinitybotlist/eureka/genconfig"
	"github.com/infinitybotlist/eureka/hotcache"
	redishc "github.com/infinitybotlist/eureka/hotcache/redis"
	"github.com/infinitybotlist/eureka/ratelimit"
	"github.com/infinitybotlist/eureka/snippets"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	Pool *pgxpool.Pool

	Discord *discordgo.Session

	Config *config.Config

	Logger *zap.Logger

	Validator *validator.Validate

	Redis *goredis.Client

	Cache hotcache.HotCache[string]

	Context = context.Background()
)

func Setup() error {
	genconfig.GenConfig(config.Config{})

	configBytes, err := os.ReadFile("config.yaml")
	if err != nil {
		return err
	}

	Config = &config.Config{}
	if err := yaml.Unmarshal(configBytes, Config); err != nil {
		return err
	}

	Logger = snippets.CreateZap()

	Validator = validator.New()
	Validator.RegisterValidation("httpurl", snippets.ValidatorIsHttpOrHttps)
	Validator.RegisterValidation("https", snippets.ValidatorIsHttps)
	Validator.RegisterValidation("nospaces", snippets.ValidatorNoSpaces)

	poolConfig, err := pgxpool.ParseConfig(Config.Database.PostgresURL)
	if err != nil {
		return err
	}

	maxConns := Config.Database.MaxConns
	if maxConns <= 0 {
		maxConns = 20
	}
	minConns := Config.Database.MinConns
	if minConns <= 0 {
		minConns = 2
	}
	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = minConns
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(Context, poolConfig)
	if err != nil {
		return err
	}
	Pool = pool

	if err := migrations.Apply(Context, Pool); err != nil {
		return err
	}

	if err := setupRedis(); err != nil {
		return err
	}

	return nil
}

func setupRedis() error {
	if Config.Redis.URL == "" {
		Logger.Warn("no redis URL configured; running without response caching or rate limiting")
		return nil
	}

	opts, err := goredis.ParseURL(Config.Redis.URL)
	if err != nil {
		return err
	}
	Redis = goredis.NewClient(opts)

	if err := Redis.Ping(Context).Err(); err != nil {
		return err
	}

	Cache = redishc.RedisHotCache[string]{Redis: Redis, Prefix: "cache:"}

	ratelimit.SetupState(&ratelimit.RLState{
		HotCache: redishc.RedisHotCache[int]{Redis: Redis, Prefix: "ratelimit:"},
	})

	return nil
}
