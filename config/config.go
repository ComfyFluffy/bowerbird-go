package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

// The version of config and app
const (
	Version   = "0.1.0"
	UIVersion = ""
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

//Config defines the config stuct of this app.
type Config struct {
	Log      LogConfig
	Server   ServerConfig
	Storage  StorageConfig
	Database DatabaseConfig
	Network  NetworkConfig
	System   SystemConfig

	Pixiv PixivConfig

	Path string `json:"-"`

	encoder *json.Encoder
	buf     *bytes.Buffer
}

// LogConfig defines the Log field in Config.
type LogConfig struct {
	ConsoleLevel string
	FileLevel    string
	File         string
}

// StorageConfig defines the Storage field in Config
type StorageConfig struct {
	RootDir string
	Pixiv   string
}

// ParsedPixiv returns the Storage.Pixiv if it is absolute path,
// otherwise it returns "Storage.RootDir/Storage.Pixiv".
func (s *StorageConfig) ParsedPixiv() string {
	return join(s.RootDir, s.Pixiv)
}

// ServerConfig defines the Server field in Config.
type ServerConfig struct {
	Address string
}

// DatabaseConfig defines the Database field in Config.
type DatabaseConfig struct {
	Enabled      bool
	MongoURI     string
	DatabaseName string
}

// PixivConfig defines the Pixiv field in Config.
type PixivConfig struct {
	RefreshToken    string
	Language        string
	APIProxy        string
	DownloaderProxy string
}

// NetworkConfig defines the Network field in Config.
type NetworkConfig struct {
	GlobalProxy string
}

// SystemConfig defines the System field in Config.
type SystemConfig struct {
	FFmpegCommand string
}

// Load loads the Config from the given json bytes.
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

// Marshal returns the json bytes of Config.
func (c *Config) Marshal() ([]byte, error) {
	defer c.buf.Reset()
	err := c.encoder.Encode(c)
	return c.buf.Bytes(), err
}

// Save writes the json bytes of Config into the c.Path
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

// New builds a *Config with default values.
func New() *Config {
	buf := &bytes.Buffer{}
	m := json.NewEncoder(buf)
	m.SetIndent("", "    ")
	m.SetEscapeHTML(false)
	return &Config{
		encoder: m,
		buf:     buf,
		Log: LogConfig{
			File:         "log/bowerbird.log",
			ConsoleLevel: "INFO",
			FileLevel:    "INFO",
		},
		Server: ServerConfig{
			Address: "127.0.0.1:10233",
		},
		Storage: StorageConfig{
			RootDir: defaultRoot,
			Pixiv:   "pixiv",
		},
		Database: DatabaseConfig{
			// MongoURI referennce: https://docs.mongodb.com/manual/reference/connection-string/
			MongoURI:     "mongodb://localhost",
			DatabaseName: "bowerbird",
		},
		Pixiv: PixivConfig{
			Language: "en",
		},
		System: SystemConfig{
			FFmpegCommand: "ffmpeg",
		},
	}
}
