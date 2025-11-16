package config

import (
	"laguna/utils"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Engine    Engine    `yaml:"engine"`
	Logger    Logger    `yaml:"logger"`
	Transport Transport `yaml:"transport"`
	WAL       WAL       `yaml:"wal"`
}

type Engine struct {
	Type string `yaml:"type"`
}

type Logger struct {
	Level             string `yaml:"level"`
	Out               string `yaml:"out"`
	DisableStacktrace bool   `yaml:"disable_stacktrace"`
}

type Transport struct {
	Type           string         `yaml:"type"`
	Address        string         `yaml:"address"`
	MaxConn        int64          `yaml:"max_conn"`
	MaxMessageSize utils.ByteSize `yaml:"max_message_size"`
	IdleTimeout    time.Duration  `yaml:"idle_timeout"`
	SemWaitTimeout time.Duration  `yaml:"sem_wait_timeout"`
}

type WAL struct {
	FlushInterval  time.Duration  `yaml:"flush_interval"`
	FlushBatchSize int64          `yaml:"flush_batch_size"`
	MaxSegmentSize utils.ByteSize `yaml:"max_segment_size"`
	Directory      string         `yaml:"directory"`
}

func NewConfig(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Printf("config inited: %+v", config)
	return &config
}
