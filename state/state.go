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
	"github.com/infinitybotlist/eureka/snippets"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	Pool *pgxpool.Pool

	Discord *discordgo.Session

	Config *config.Config

	Logger *zap.Logger

	Validator *validator.Validate

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
	// Explicit sizing instead of pgx's defaults (MaxConns = 4x CPU cores,
	// MinConns = 0): a MinConns floor keeps connections warm so a burst of
	// traffic after idle time doesn't pay full connection-establishment
	// latency, and MaxConnLifetime/MaxConnIdleTime recycle connections
	// instead of holding them open indefinitely.
	//
	// The struct tag defaults below (20/2) only apply to the generated
	// config.yaml.sample, not to an existing config.yaml that predates
	// these fields — falling back here too means a config.yaml missing
	// them gets the same sane pool instead of pgxpool panicking on
	// MaxConns=0 ("MaxSize must be >= 1").
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

	return nil
}
