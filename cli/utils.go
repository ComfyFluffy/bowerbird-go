package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/WOo0W/bowerbird/cli/color"
	"github.com/WOo0W/bowerbird/cli/log"
	"github.com/WOo0W/bowerbird/config"
	"github.com/WOo0W/bowerbird/downloader"
	"github.com/WOo0W/go-pixiv/pixiv"
	"github.com/dustin/go-humanize"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	printGetPixivUsername = "pixiv Username / Email: "
)

func loadConfigFile(ctx context.Context, conf *config.Config, path string) error {
	logger := log.FromContext(ctx)
	if path == "" {
		if err := os.MkdirAll(conf.Storage.RootDir, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		path = filepath.Join(conf.Storage.RootDir, "config.json")
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info(fmt.Sprintf("Creating new config: %q", path))
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

	conf.Path = path
	err = conf.Save()
	if err != nil {
		logger.Warn("Cannot save config file:", err)
	}

	logger.ConsoleLevel = log.ParseLevel(conf.Log.ConsoleLevel)
	logger.FileLevel = log.ParseLevel(conf.Log.FileLevel)
	return nil
}

func getUserPass(printUser string) (username, password string) {
	fmt.Print(printUser)
	fmt.Scanln(&username)

	fmt.Print("Password: ")
	b, _ := terminal.ReadPassword(int(syscall.Stdin))
	password = string(b)

	return username, password
}

// authPixiv tries to auth to pixiv.
// If RefreshToken is empty, it will ask for
// pixiv username and password.
func authPixiv(api *pixiv.AppAPI, c *config.Config) error {
	if c.Pixiv.RefreshToken == "" {
		u, p := getUserPass(printGetPixivUsername)
		api.SetUser(u, p)
		_, err := api.ForceAuth()
		if err != nil {
			return err
		}
	} else {
		api.SetRefreshToken(c.Pixiv.RefreshToken)
		_, err := api.ForceAuth()
		if pixiv.IsInvalidCredentials(err) {
			u, p := getUserPass(printGetPixivUsername)
			fmt.Println(u, p)
			api.SetUser(u, p)
			_, err := api.ForceAuth()
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

// downloaderUILoop displays current download speed
func downloaderUILoop(dl *downloader.Downloader) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Printf("[%s/s]     \r", color.SHiGreen(humanize.Bytes(uint64(dl.BytesLastSec))))
			}
		}
	}()
	dl.Wait()
}

func connectToDB(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return client, err
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return client, err
	}
	return client, nil
}

func getPixivUserFlag(c *cli.Context, fallbackID int) (id int) {
	if c.IsSet("user") {
		id = c.Int("user")
	} else {
		id = fallbackID
	}
	return id
}
