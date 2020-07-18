package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

var (
	userHome, _ = os.UserHomeDir()
	defaultRoot = filepath.Join(userHome, ".bowerbird")
)

type Config struct {
	Log      LogConfig      `json:"log"`
	Server   ServerConfig   `json:"server"`
	Storage  StorageConfig  `json:"storage"`
	Database DatabaseConfig `json:"database"`
	Network  NetworkConfig  `json:"network"`

	Pixiv PixivConfig `json:"pixiv"`

	Path string `json:"-"`

	encoder *json.Encoder
	buf     *bytes.Buffer
}

type LogConfig struct {
	Level string `json:"level"`
	File  string `json:"file"`
}
type StorageConfig struct {
	RootDir string `json:"rootDir"`
	Pixiv   string `json:"pixiv"`
}
type ServerConfig struct {
	IP   string `json:"ip"`
	Port uint16 `json:"port"`
}
type DatabaseConfig struct {
	MongoURI      string        `json:"mongoURI"`
	DatabaseName  string        `json:"databaseName"`
	Timeout       string        `json:"timeout"`
	TimeoutParsed time.Duration `json:"-"`
}
type ScheduleConfig struct {
}

type PixivConfig struct {
	RefreshToken string `json:"refreshToken"`
}

type NetworkConfig struct {
	Proxy string `json:"proxy"`
}

func (c *Config) Load(b []byte) error {
	err := json.Unmarshal(b, c)
	if err != nil {
		return err
	}
	c.Database.TimeoutParsed, err = time.ParseDuration(c.Database.Timeout)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) Marshal() ([]byte, error) {
	defer c.buf.Reset()
	err := c.encoder.Encode(c)
	return c.buf.Bytes(), err
}

func (c *Config) Save() error {
	if c.Path == "" {
		return errors.New("config save: no file specified")
	}
	b, err := c.Marshal()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.Path, b, 0644)
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
		Database: DatabaseConfig{
			// https://docs.mongodb.com/manual/reference/connection-string/
			MongoURI:      "mongodb://localhost",
			DatabaseName:  "bowerbird",
			Timeout:       "15s",
			TimeoutParsed: 15 * time.Second,
		},
	}
}
