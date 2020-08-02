package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/WOo0W/bowerbird/cli/log"
)

var (
	userHome, _ = os.UserHomeDir()
	defaultRoot = filepath.Join(userHome, ".bowerbird")
)

func join(root, sub string) string {
	if filepath.IsAbs(sub) {
		return sub
	}
	return filepath.Join(root, sub)
}

//Config of this program
type Config struct {
	Log      LogConfig
	Server   ServerConfig
	Storage  StorageConfig
	Database DatabaseConfig
	Network  NetworkConfig

	Pixiv PixivConfig

	Path string `json:"-"`

	encoder *json.Encoder
	buf     *bytes.Buffer
}

type LogConfig struct {
	ConsoleLevel string
	FileLevel    string
	File         string
}
type StorageConfig struct {
	RootDir string
	Pixiv   string
}

func (s *StorageConfig) ParsedPixiv() string {
	return join(s.RootDir, s.Pixiv)
}

type ServerConfig struct {
	IP   string
	Port uint16
}
type DatabaseConfig struct {
	MongoURI     string
	DatabaseName string
}

type PixivConfig struct {
	RefreshToken    string
	Language        string
	APIProxy        string
	DownloaderProxy string
}

type NetworkConfig struct {
	GlobalProxy                string
	RetryTimesMax              int
	RetryWaitMax, RetryWaitMin string
}

func (nc *NetworkConfig) RetryWaitParsed() (max, min time.Duration) {
	var err error
	max, err = time.ParseDuration(nc.RetryWaitMax)
	if err != nil {
		max = 30 * time.Second
		log.G.Warn(fmt.Sprintf("Can not parse RetryWaitMax of NetworkConfig: %s, use default: 30s", nc.RetryWaitMax))
	}
	min, err = time.ParseDuration(nc.RetryWaitMin)
	if err != nil {
		min = 1 * time.Second
		log.G.Warn(fmt.Sprintf("Can not parse RetryWaitMin of NetworkConfig: %s, use default: 1s", nc.RetryWaitMin))
	}
	return max, min
}

func (c *Config) Load(b []byte) error {
	err := json.Unmarshal(b, c)
	if err != nil {
		return err
	}
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
			File:         "log/bowerbird.log",
			ConsoleLevel: "INFO",
			FileLevel:    "INFO",
		},
		Server: ServerConfig{
			IP:   "127.0.0.1",
			Port: 10233,
		},
		Storage: StorageConfig{
			RootDir: defaultRoot,
			Pixiv:   "pixiv",
		},
		Database: DatabaseConfig{
			// https://docs.mongodb.com/manual/reference/connection-string/
			MongoURI:     "mongodb://localhost",
			DatabaseName: "bowerbird",
		},
		Network: NetworkConfig{
			RetryWaitMax:  "10s",
			RetryWaitMin:  "1s",
			RetryTimesMax: 5,
		},
		Pixiv: PixivConfig{
			Language: "en",
		},
	}
}
