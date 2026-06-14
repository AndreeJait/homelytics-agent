package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/spf13/viper"
)

// AppConfig holds all application configuration.
type AppConfig struct {
	App struct {
		Name     string `mapstructure:"name"`
		Env      string `mapstructure:"env"`
		HTTPPort int    `mapstructure:"http_port"`
	} `mapstructure:"app"`

	HTTP struct {
		Engine        string `mapstructure:"engine"`
		EnableSwagger bool   `mapstructure:"enable_swagger"`
		DebugMode     bool   `mapstructure:"debug_mode"`
	} `mapstructure:"http"`

	Log struct {
		Level       string         `mapstructure:"level"`
		Format      logw.LogFormat `mapstructure:"format"`
		WriteToFile bool           `mapstructure:"write_to_file"`
		FilePath    string         `mapstructure:"file_path"`
	} `mapstructure:"log"`

	DB struct {
		Driver          string        `mapstructure:"driver"`
		Dialect         string        `mapstructure:"dialect"`
		DSN             string        `mapstructure:"dsn"`
		MaxOpenConns    int           `mapstructure:"max_open_conns"`
		MaxIdleConns    int           `mapstructure:"max_idle_conns"`
		ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
		DebugMode       bool          `mapstructure:"debug_mode"`
	} `mapstructure:"db"`

	Redis struct {
		Address   string `mapstructure:"address"`
		Password  string `mapstructure:"password"`
		DB        int    `mapstructure:"db"`
		PoolSize  int    `mapstructure:"pool_size"`
		DebugMode bool   `mapstructure:"debug_mode"`
	} `mapstructure:"redis"`

	Graceful struct {
		ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	} `mapstructure:"graceful"`

	Daemon struct {
		RunDir    string `mapstructure:"run_dir"`
		EtcDir    string `mapstructure:"etc_dir"`
		LogDir    string `mapstructure:"log_dir"`
		StateFile string `mapstructure:"state_file"`
	} `mapstructure:"daemon"`

	IPC struct {
		SocketPath string `mapstructure:"socket_path"`
	} `mapstructure:"ipc"`

	Homelytics struct {
		MockMode           bool   `mapstructure:"mock_mode"`
		BaseURL            string `mapstructure:"base_url"`
		MockTSNetAuthKey   string `mapstructure:"mock_tsnet_auth_key"`
	} `mapstructure:"homelytics"`

	TSNet struct {
		Hostname              string   `mapstructure:"hostname"`
		ControlURL            string   `mapstructure:"control_url"`
		AdvertiseTags         []string `mapstructure:"advertise_tags"`
		Dir                   string   `mapstructure:"dir"`
		EnableCommandListener bool     `mapstructure:"enable_command_listener"`
		CommandListenerAddr   string   `mapstructure:"command_listener_addr"`
	} `mapstructure:"tsnet"`

	Containerd struct {
		Address   string        `mapstructure:"address"`
		Namespace string        `mapstructure:"namespace"`
		Timeout   time.Duration `mapstructure:"timeout"`
	} `mapstructure:"containerd"`

	Heartbeat struct {
		Enabled  bool          `mapstructure:"enabled"`
		Interval time.Duration `mapstructure:"interval"`
	} `mapstructure:"heartbeat"`
}

// Load reads the base config file at configPath, then merges app.local.yaml
// from the same directory if it exists. Environment variables override both.
func Load(configPath string) (*AppConfig, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Merge local overrides if app.local.yaml exists alongside the base config
	localPath := strings.Replace(configPath, "app.yaml", "app.local.yaml", 1)
	if _, err := os.Stat(localPath); err == nil {
		v.SetConfigFile(localPath)
		if err := v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("failed to load local config: %w", err)
		}
	}

	cfg := &AppConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults
	if cfg.HTTP.Engine == "" {
		cfg.HTTP.Engine = "echo"
	}
	if cfg.App.HTTPPort == 0 {
		cfg.App.HTTPPort = 8080
	}
	if cfg.DB.Driver == "" {
		cfg.DB.Driver = "gorm"
	}
	if cfg.DB.Dialect == "" {
		cfg.DB.Dialect = "postgres"
	}
	if cfg.Graceful.ShutdownTimeout == 0 {
		cfg.Graceful.ShutdownTimeout = 10 * time.Second
	}
	if cfg.Daemon.RunDir == "" {
		cfg.Daemon.RunDir = "/opt/homelytics/run"
	}
	if cfg.Daemon.EtcDir == "" {
		cfg.Daemon.EtcDir = "/opt/homelytics/etc"
	}
	if cfg.Daemon.LogDir == "" {
		cfg.Daemon.LogDir = "/opt/homelytics/log"
	}
	if cfg.Daemon.StateFile == "" {
		cfg.Daemon.StateFile = "/opt/homelytics/etc/state.json"
	}
	if cfg.IPC.SocketPath == "" {
		cfg.IPC.SocketPath = "/opt/homelytics/run/ipc.sock"
	}
	if cfg.Homelytics.BaseURL == "" {
		cfg.Homelytics.BaseURL = "https://api.homelytics.internal"
	}
	if cfg.TSNet.Hostname == "" {
		cfg.TSNet.Hostname = "homelytics-agent"
	}
	if cfg.TSNet.CommandListenerAddr == "" {
		cfg.TSNet.CommandListenerAddr = ":7373"
	}
	if cfg.Containerd.Address == "" {
		cfg.Containerd.Address = "/run/containerd/containerd.sock"
	}
	if cfg.Containerd.Namespace == "" {
		cfg.Containerd.Namespace = "homelytics"
	}
	if cfg.Containerd.Timeout == 0 {
		cfg.Containerd.Timeout = 10 * time.Second
	}
	if cfg.Heartbeat.Interval == 0 {
		cfg.Heartbeat.Interval = 30 * time.Second
	}

	return cfg, nil
}
