package state

import (
	"context"
	"os"

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

	pool, err := pgxpool.New(Context, Config.PostgresURL)
	if err != nil {
		return err
	}
	Pool = pool

	if err := migrations.Apply(Context, Pool); err != nil {
		return err
	}

	return nil
}
