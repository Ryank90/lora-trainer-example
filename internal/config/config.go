package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Redis        RedisConfig        `yaml:"redis"`
	Storage      StorageConfig      `yaml:"storage"`
	Providers    ProvidersConfig    `yaml:"providers"`
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	Telemetry    TelemetryConfig    `yaml:"telemetry"`
	Auth         AuthConfig         `yaml:"auth"`
}

type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
	MaxConns int    `yaml:"max_conns"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type StorageConfig struct {
	Endpoint        string `yaml:"endpoint"`
	Region          string `yaml:"region"`
	Bucket          string `yaml:"bucket"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey  string `yaml:"secret_access_key"`
	PresignDuration time.Duration `yaml:"presign_duration"`
}

type ProvidersConfig struct {
	VoltagePark VoltageParkConfig `yaml:"voltagepark"`
	Runpod      RunpodConfig      `yaml:"runpod"`
	Preference  []string          `yaml:"preference"`
}

type VoltageParkConfig struct {
	Enabled  bool   `yaml:"enabled"`
	APIKey   string `yaml:"api_key"`
	Endpoint string `yaml:"endpoint"`
}

type RunpodConfig struct {
	Enabled  bool   `yaml:"enabled"`
	APIKey   string `yaml:"api_key"`
	Endpoint string `yaml:"endpoint"`
}

type OrchestratorConfig struct {
	Workers          int           `yaml:"workers"`
	PollInterval     time.Duration `yaml:"poll_interval"`
	ProvisionTimeout time.Duration `yaml:"provision_timeout"`
	DownloadTimeout  time.Duration `yaml:"download_timeout"`
	UploadTimeout    time.Duration `yaml:"upload_timeout"`
	MaxRetries       int           `yaml:"max_retries"`
	WarmPool         WarmPoolConfig `yaml:"warm_pool"`
}

type WarmPoolConfig struct {
	Enabled     bool          `yaml:"enabled"`
	MinSize     int           `yaml:"min_size"`
	MaxSize     int           `yaml:"max_size"`
	MaxIdleTime time.Duration `yaml:"max_idle_time"`
}

type TelemetryConfig struct {
	MetricsEnabled bool   `yaml:"metrics_enabled"`
	MetricsPath    string `yaml:"metrics_path"`
	TracingEnabled bool   `yaml:"tracing_enabled"`
	OTLPEndpoint   string `yaml:"otlp_endpoint"`
	ServiceName    string `yaml:"service_name"`
}

type AuthConfig struct {
	APIKeys []string `yaml:"api_keys"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	data = []byte(os.ExpandEnv(string(data)))

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.applyDefaults()

	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Server.ShutdownTimeout == 0 {
		c.Server.ShutdownTimeout = 15 * time.Second
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.MaxConns == 0 {
		c.Database.MaxConns = 20
	}
	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}
	if c.Storage.Region == "" {
		c.Storage.Region = "auto"
	}
	if c.Storage.PresignDuration == 0 {
		c.Storage.PresignDuration = 1 * time.Hour
	}
	if c.Orchestrator.Workers == 0 {
		c.Orchestrator.Workers = 2
	}
	if c.Orchestrator.PollInterval == 0 {
		c.Orchestrator.PollInterval = 5 * time.Second
	}
	if c.Orchestrator.ProvisionTimeout == 0 {
		c.Orchestrator.ProvisionTimeout = 10 * time.Minute
	}
	if c.Orchestrator.DownloadTimeout == 0 {
		c.Orchestrator.DownloadTimeout = 10 * time.Minute
	}
	if c.Orchestrator.UploadTimeout == 0 {
		c.Orchestrator.UploadTimeout = 10 * time.Minute
	}
	if c.Orchestrator.MaxRetries == 0 {
		c.Orchestrator.MaxRetries = 2
	}
	if c.Telemetry.MetricsPath == "" {
		c.Telemetry.MetricsPath = "/metrics"
	}
	if c.Telemetry.ServiceName == "" {
		c.Telemetry.ServiceName = "lora-trainer"
	}
}
