package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/WOo0W/bowerbird/cli/color"

	"github.com/WOo0W/bowerbird/cli/log"

	"github.com/WOo0W/bowerbird/config"
	"github.com/urfave/cli/v2"
)

func loadConfigFile(conf *config.Config, path string) error {
	var isConfigFileSet bool
	if path == "" {
		isConfigFileSet = false
		if err := os.MkdirAll(conf.Storage.RootDir, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		path = filepath.Join(conf.Storage.RootDir, "config.json")
		log.G.Notice("Flag config not set, use default file:", path)
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if isConfigFileSet {
				return err
			}
			if b, err = conf.Marshal(); err != nil {
				return err
			}
			return ioutil.WriteFile(path, b, 0644)
		}
		return err
	}
	if err = conf.Load(b); err != nil {
		return err
	}
	if b, err = conf.Marshal(); err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

func New() *cli.App {
	conf := config.New()
	configFile := ""

	return &cli.App{
		Writer:    color.Stdout,
		ErrWriter: color.Stderr,
		Name:      "Bowerbird",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "The path of JSON config file",
				Destination: &configFile,
			},
		},
		// Load and save config file
		Before: func(c *cli.Context) error {
			err := loadConfigFile(conf, configFile)

			if err != nil {
				// prevent help message and error message printing to Stdout
				c.App.Writer = ioutil.Discard
				return cli.Exit(color.SHiRed("Error while loading config: ", err), 1)
			}
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Action: func(c *cli.Context) error {
					println("Running server...")
					time.Sleep(3 * time.Second)
					return nil
				},
			},
		},
	}
}
