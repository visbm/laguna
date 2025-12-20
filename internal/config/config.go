package config

import (
	"laguna/utils/data_type"
	"laguna/utils/fs"
	"log"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Engine      Engine      `yaml:"engine"`
	Logger      Logger      `yaml:"logger"`
	Transport   Transport   `yaml:"transport"`
	WAL         WAL         `yaml:"wal"`
	Replication Replication `yaml:"replication"`
	Profiler    Profiler    `yaml:"profiler"`
}

type Engine struct {
	Type   string `yaml:"type"`
	Shards int64  `yaml:"shards"`
}

type Logger struct {
	Level             string `yaml:"level"`
	Out               string `yaml:"out"`
	DisableStacktrace bool   `yaml:"disable_stacktrace"`
}

type Transport struct {
	Type  string    `yaml:"type"`
	Sever TCPServer `yaml:"server"`
}

type WAL struct {
	Enable         bool               `yaml:"enable"`
	FlushInterval  time.Duration      `yaml:"flush_interval"`
	FlushBatchSize int64              `yaml:"flush_batch_size"`
	MaxSegmentSize data_type.ByteSize `yaml:"max_segment_size"`
	Directory      string             `yaml:"directory"`
}

type Replication struct {
	Enable       bool          `yaml:"enable"`
	IsMaster     bool          `yaml:"is_master"`
	SyncInterval time.Duration `yaml:"sync_interval"`
	Client       Client        `yaml:"client"`
	Server       TCPServer     `yaml:"server"`
}

type Client struct {
	Address     string             `yaml:"address"`
	MaxRespSize data_type.ByteSize `yaml:"max_resp_size"`
	Deadline    time.Duration      `yaml:"deadline"`
	RetryCount  int64              `yaml:"retry_count"`
}
type TCPServer struct {
	Address        string             `yaml:"address"`
	MaxConn        int64              `yaml:"max_conn"`
	MaxMessageSize data_type.ByteSize `yaml:"max_message_size"`
	IdleTimeout    time.Duration      `yaml:"idle_timeout"`
	SemWaitTimeout time.Duration      `yaml:"sem_wait_timeout"`
}

type Profiler struct {
	Enable  bool   `yaml:"enable"`
	Address string `yaml:"address"`
}

func NewConfig(path string) *Config {
	data, err := fs.ReadFile(path)
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
