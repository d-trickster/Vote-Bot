package config

import (
	"flag"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Env     string `yaml:"env"`
	LogPath string `yaml:"log_path"`
	Storage string `yaml:"storage"`

	MainChatId int64   `yaml:"main_chat_id"`
	Admins     []int64 `yaml:"admin"`

	FetchInterval time.Duration `yaml:"polling_interval"`

	Limit  int `yaml:"limit"`
	Offset int `yaml:"offset"`
}

func MustLoad() *Config {
	path := fetchConfigPath()
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var cfg Config

	if err = yaml.NewDecoder(f).Decode(&cfg); err != nil {
		panic(err)
	}

	return &cfg
}

// Priority:
// flag > env > default=""
func fetchConfigPath() string {
	var path string

	flag.StringVar(&path, "config", "", "path to config.yaml")
	flag.Parse()

	if path == "" {
		path = os.Getenv("VOTE_CONFIG_PATH")
	}

	return path
}
