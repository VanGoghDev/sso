package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env            string             `yaml:"env" env-default:"local"`
	StoragePath    string             `yaml:"storage_path" env-required:"true"`
	GRPC           GRPCConfig         `yaml:"grpc"`
	EmailService   EmailSenderConfig  `yaml:"emailSender"`
	Verification   VerificationConfig `yaml:"verification"`
	MigrationsPath string             `yaml:"migrations_path"`
	TokenTTL       time.Duration      `yaml:"token_ttl" env-default:"1h"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type EmailSenderConfig struct {
	Name     string `yaml:"name"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

type VerificationConfig struct {
	Len       int `yaml:"len"`
	LastHours int `yaml:"hours"`
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	return MustLoadPath(configPath)
}

func MustLoadPath(configPath string) *Config {
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	return &cfg
}

// fetchConfigPath fetches config path from command line flag or environment variable.
// Priority: flag > env > default.
// Default value is empty string.
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
