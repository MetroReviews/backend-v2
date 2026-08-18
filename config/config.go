package config

import "strconv"

type Config struct {
	Discord  Discord  `yaml:"discord"`
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Auth     Auth     `yaml:"auth"`
}

type Discord struct {
	ClientID     string `yaml:"client_id" default:"" comment:"Discord OAuth2 client ID"`
	ClientSecret string `yaml:"client_secret" default:"" comment:"Discord OAuth2 client secret"`
	Token        string `yaml:"token" default:"" comment:"Discord bot token"`

	StaffServerID  string `yaml:"staff_server_id" default:"" comment:"Staff server ID"`
	QueueChannelID string `yaml:"queue_channel_id" default:"" comment:"Queue channel ID"`
	LogsChannelID  string `yaml:"logs_channel_id" default:"" comment:"Logs channel ID"`

	Owners []int64 `yaml:"owners" default:"" comment:"Owner user IDs"`
}

type Server struct {
	BindAddr string `yaml:"bind_addr" default:":8080" comment:"e.g. :8080"`
}

type Auth struct {
	AllowedHost       string `yaml:"allowed_host" default:"" comment:"admin/host, e.g. catnip.metrobots.xyz" required:"false"`
	OAuthRedirect     string `yaml:"oauth_redirect" default:"" comment:"frostpaw redirect URI" required:"false"`
	DefaultLoginState string `yaml:"default_login_state" default:"" comment:"fallback panel origin" required:"false"`
}

type Database struct {
	PostgresURL string `yaml:"postgres_url" default:"postgres:///brc" comment:"e.g. postgres:///brc"`
	MaxConns    int32  `yaml:"max_conns" default:"20" comment:"Max open connections in the Postgres pool"`
	MinConns    int32  `yaml:"min_conns" default:"2" comment:"Min idle connections kept warm in the Postgres pool"`
}

func parseID(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

func (c *Config) GuildID() uint64 { return parseID(c.Discord.StaffServerID) }

func (c *Config) QueueChannelID() uint64 { return parseID(c.Discord.QueueChannelID) }

func (c *Config) LogsChannelID() uint64 { return parseID(c.Discord.LogsChannelID) }

func (c *Config) IsOwner(id int64) bool {
	for _, o := range c.Discord.Owners {
		if o == id {
			return true
		}
	}
	return false
}
