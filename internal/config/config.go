package config

import (
	"fmt"
	"log/slog"
	"time"
	"xmine/litebans-api/internal/domain"
	"xmine/litebans-api/internal/logging"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Ip               string `env:"BUILD_TARGET,required"`
	DatabaseDriver   string `env:"DATABASE_DRIVER,required"`
	DatabaseHost     string `env:"DATABASE_HOST,required"`
	DatabasePort     string `env:"DATABASE_PORT,required"`
	DatabaseUser     string `env:"DATABASE_USER,required"`
	DatabasePassword string `env:"DATABASE_PASSWORD,required"`
	DatabaseName     string `env:"DATABASE_NAME,required"`

	HTTPAddr  string            `env:"HTTP_ADDR" envDefault:":8080"`
	LogFormat logging.LogFormat `env:"LOG_FORMAT" envDefault:"text"`
	LogLevel  slog.Level        `env:"LOG_LEVEL" envDefault:"info"`

	// TablePrefix is the LiteBans table name prefix (e.g. "litebans_bans").
	TablePrefix string `env:"TABLE_PREFIX" envDefault:"litebans_"`
	// ConsoleAliases are moderator names that denote the server console rather than a player.
	ConsoleAliases []string `env:"CONSOLE_ALIASES" envSeparator:"," envDefault:"CONSOLE,Console"`

	// IncludeInactiveDefault/IncludeSilentDefault are deployment-wide defaults for list endpoint visibility (2.4 TOR).
	IncludeInactiveDefault bool `env:"INCLUDE_INACTIVE" envDefault:"true"`
	IncludeSilentDefault   bool `env:"INCLUDE_SILENT" envDefault:"true"`

	DefaultPageSize int `env:"DEFAULT_PAGE_SIZE" envDefault:"10"`
	MaxPageSize     int `env:"MAX_PAGE_SIZE" envDefault:"100"`

	ObfuscateIDs      bool   `env:"OBFUSCATE_IDS" envDefault:"false"`
	ObfuscationSecret string `env:"OBFUSCATION_SECRET" envDefault:""`

	JWTPublicKeyPath  string        `env:"JWT_PUBLIC_KEY_PATH,required"`
	JWTIssuer         string        `env:"JWT_ISSUER" envDefault:"xmine-identity"`
	AuthorityAPIURL   string        `env:"AUTHORITY_API_URL,required"`
	InternalToken     string        `env:"INTERNAL_TOKEN,required"`
	ModPermission     string        `env:"MOD_PERMISSION" envDefault:"web.litebans.view.all"`
	PublicTypes       []string      `env:"PUBLIC_TYPES" envSeparator:"," envDefault:"ban"`
	AuthorityCacheTTL time.Duration `env:"AUTHORITY_CACHE_TTL" envDefault:"60s"`
}

func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return c, err
	}
	if err := c.validate(); err != nil {
		return c, err
	}
	return c, nil
}

func (c Config) validate() error {
	if c.DatabaseDriver != "mysql" {
		return fmt.Errorf("unsupported DATABASE_DRIVER %q: only \"mysql\" is supported", c.DatabaseDriver)
	}
	if c.ObfuscateIDs && c.ObfuscationSecret == "" {
		return fmt.Errorf("OBFUSCATION_SECRET is required when OBFUSCATE_IDS=true")
	}
	if c.DefaultPageSize <= 0 {
		return fmt.Errorf("DEFAULT_PAGE_SIZE must be positive")
	}
	if c.MaxPageSize <= 0 || c.MaxPageSize < c.DefaultPageSize {
		return fmt.Errorf("MAX_PAGE_SIZE must be positive and >= DEFAULT_PAGE_SIZE")
	}
	if len(c.PublicTypes) == 0 {
		return fmt.Errorf("PUBLIC_TYPES must not be empty")
	}
	for _, pt := range c.PublicTypes {
		if !isValidPunishmentTypeName(pt) {
			return fmt.Errorf("PUBLIC_TYPES contains invalid type %q: must be one of ban, mute, warning, kick", pt)
		}
	}
	return nil
}

func isValidPunishmentTypeName(name string) bool {
	switch domain.PunishmentType(name) {
	case domain.TypeBan, domain.TypeMute, domain.TypeWarning, domain.TypeKick:
		return true
	default:
		return false
	}
}
