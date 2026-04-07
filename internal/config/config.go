package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Profile selects which collectors run.
type Profile string

const (
	ProfileSafe       Profile = "safe"
	ProfileExtended   Profile = "extended"
	ProfileAggressive Profile = "aggressive"
)

// Config holds exporter and ClickHouse client options.
type Config struct {
	ListenAddress string `yaml:"listen_address"`

	// ClickHouse DSN or discrete fields (DSN takes precedence if set).
	DSN      string `yaml:"dsn"`
	Address  string `yaml:"address"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	TLS TLS `yaml:"tls"`

	Profile Profile `yaml:"profile"`

	CollectTimeout time.Duration `yaml:"collect_timeout"`
	QueryTimeout   time.Duration `yaml:"query_timeout"`
	MaxOpenConns   int           `yaml:"max_open_conns"`

	// PartsTopN limits aggressive collector labels (database, table pairs).
	PartsTopN int `yaml:"parts_top_n"`
}

// TLS optional client settings.
type TLS struct {
	Enabled    bool   `yaml:"enabled"`
	CAFile     string `yaml:"ca_file"`
	ServerName string `yaml:"server_name"`
	Insecure   bool   `yaml:"insecure_skip_verify"`
}

// Default returns sensible defaults.
func Default() *Config {
	return &Config{
		ListenAddress:    ":9101",
		Database:         "default",
		Profile:          ProfileSafe,
		CollectTimeout:   25 * time.Second,
		QueryTimeout:     20 * time.Second,
		MaxOpenConns:     4,
		PartsTopN:        20,
	}
}

// LoadFile merges YAML file into base config.
func LoadFile(path string, base *Config) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, base); err != nil {
		return nil, err
	}
	return base, nil
}

// ApplyEnv overrides from CH_EXPORTER_* variables.
func ApplyEnv(c *Config) {
	set := func(env string, dst *string) {
		if v := os.Getenv(env); v != "" {
			*dst = v
		}
	}
	set("CH_EXPORTER_LISTEN_ADDRESS", &c.ListenAddress)
	set("CH_EXPORTER_DSN", &c.DSN)
	set("CH_EXPORTER_ADDRESS", &c.Address)
	set("CH_EXPORTER_DATABASE", &c.Database)
	set("CH_EXPORTER_USERNAME", &c.Username)
	set("CH_EXPORTER_PASSWORD", &c.Password)
	if v := os.Getenv("CH_EXPORTER_PROFILE"); v != "" {
		c.Profile = Profile(v)
	}
	if v := os.Getenv("CH_EXPORTER_TLS_CA_FILE"); v != "" {
		c.TLS.CAFile = v
		c.TLS.Enabled = true
	}
	if v := os.Getenv("CH_EXPORTER_TLS_SERVER_NAME"); v != "" {
		c.TLS.ServerName = v
	}
	if v := os.Getenv("CH_EXPORTER_TLS_INSECURE_SKIP_VERIFY"); v != "" {
		c.TLS.Insecure = strings.EqualFold(v, "true") || v == "1"
		if c.TLS.Insecure {
			c.TLS.Enabled = true
		}
	}
	if v := os.Getenv("CH_EXPORTER_PARTS_TOP_N"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.PartsTopN = n
		}
	}
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if c.DSN == "" && c.Address == "" {
		return fmt.Errorf("either dsn or address is required")
	}
	switch c.Profile {
	case ProfileSafe, ProfileExtended, ProfileAggressive:
	default:
		return fmt.Errorf("unknown profile: %q (use safe, extended, aggressive)", c.Profile)
	}
	return nil
}
