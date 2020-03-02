package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

var (
	userHome, _ = os.UserHomeDir()
	defaultRoot = filepath.Join(userHome, ".bowerbird")
)

type Config struct {
	Log     LogConfig     `json:"log"`
	Server  ServerConfig  `json:"server"`
	Storage StorageConfig `json:"storage"`

	encoder *json.Encoder
	buf     *bytes.Buffer
}

type LogConfig struct {
	Level     string `json:"level"`
	File      string `json:"file"`
	UseStdout bool   `json:"useStdout"`
}
type StorageConfig struct {
	RootDir string `json:"rootDir"`
}
type ServerConfig struct {
	IP   string `json:"ip"`
	Port uint16 `json:"port"`
}
type DatabaseConfig struct {
	UseURI   bool   `json:"useUri"`
	MongoURI string `json:"mongoUri"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func (c *Config) Load(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *Config) Marshal() ([]byte, error) {
	defer c.buf.Reset()
	err := c.encoder.Encode(c)
	return c.buf.Bytes(), err
}

func New() *Config {
	var buf bytes.Buffer
	m := json.NewEncoder(&buf)
	m.SetIndent("", "    ")
	m.SetEscapeHTML(false)
	return &Config{
		encoder: m,
		buf:     &buf,
		Log: LogConfig{
			File:  "log/bowerbird.log",
			Level: "INFO",
		},
		Server: ServerConfig{
			IP:   "127.0.0.1",
			Port: 10233,
		},
		Storage: StorageConfig{
			RootDir: defaultRoot,
		},
	}
}
