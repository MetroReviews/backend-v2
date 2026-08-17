package config

import "strconv"

type Config struct {
	Token        string  `yaml:"token" default:"" comment:"Main bot token"`
	HelperToken  string  `yaml:"helper_token" default:"" comment:"Secondary/helper bot token (unused directly, kept for parity)" required:"false"`
	GID          string  `yaml:"gid" default:"" comment:"Main guild ID"`
	Reviewer     string  `yaml:"reviewer" default:"" comment:"Reviewer role ID"`
	QueueChannel string  `yaml:"queue_channel" default:"" comment:"Channel where new bots are announced"`
	ListOwner    string  `yaml:"list_owner" default:"" comment:"List Owner role ID"`
	Sudo         string  `yaml:"sudo" default:"" comment:"Sudo role ID"`
	TestPingRole string  `yaml:"test_ping_role" default:"" comment:"Optional role pinged instead of reviewer" required:"false"`
	ClientSecret string  `yaml:"client_secret" default:"" comment:"Discord OAuth2 client secret"`
	Owners       []int64 `yaml:"owners" default:"" comment:"Bot owner user IDs"`

	PostgresURL       string `yaml:"postgres_url" default:"postgres:///brc" comment:"e.g. postgres:///brc" required:"false"`
	BindAddr          string `yaml:"bind_addr" default:":8080" comment:"e.g. :8080" required:"false"`
	AllowedHost       string `yaml:"allowed_host" default:"" comment:"admin/host, e.g. catnip.metrobots.xyz" required:"false"`
	OAuthRedirect     string `yaml:"oauth_redirect" default:"" comment:"frostpaw redirect URI" required:"false"`
	DefaultLoginState string `yaml:"default_login_state" default:"" comment:"fallback panel origin" required:"false"`
}

func parseID(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

func (c *Config) GuildID() uint64 { return parseID(c.GID) }

func (c *Config) ReviewerRole() uint64 { return parseID(c.Reviewer) }

func (c *Config) QueueChannelID() uint64 { return parseID(c.QueueChannel) }

func (c *Config) ListOwnerRole() uint64 { return parseID(c.ListOwner) }

func (c *Config) SudoRole() uint64 { return parseID(c.Sudo) }

func (c *Config) IsOwner(id int64) bool {
	for _, o := range c.Owners {
		if o == id {
			return true
		}
	}
	return false
}

func (c *Config) PingRole() string {
	if c.TestPingRole != "" {
		return c.TestPingRole
	}
	return c.Reviewer
}
