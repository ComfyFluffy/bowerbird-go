package cli

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/WOo0W/bowerbird/cli/log"
	"github.com/WOo0W/bowerbird/config"
	"github.com/WOo0W/bowerbird/downloader"
	"github.com/WOo0W/go-pixiv/pixiv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/proxy"
)

func setProxy(tr *http.Transport, uri string) error {
	pr, err := url.Parse(uri)
	if err != nil {
		return err
	}

	switch strings.ToLower(pr.Scheme) {
	case "http":
		hp := http.ProxyURL(pr)
		tr.Proxy = hp
	case "socks5":
		var spauth *proxy.Auth
		spw, _ := pr.User.Password()
		spu := pr.User.Username()
		if spw != "" || spu != "" {
			spauth = &proxy.Auth{User: spu, Password: spw}
		}
		spd, err := proxy.SOCKS5("tcp", pr.Host, spauth, proxy.Direct)
		if err != nil {
			return err
		}
		tr.DialContext = spd.(proxy.ContextDialer).DialContext
	default:
		return errors.New("set proxy: unsupported protocol")
	}
	return nil
}

func loadConfigFile(conf *config.Config, path string) error {
	if path == "" {
		if err := os.MkdirAll(conf.Storage.RootDir, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		path = filepath.Join(conf.Storage.RootDir, "config.json")
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.G.Info("Creating new config file:", path)
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
		log.G.Warn("Can not save config file:", path, "\n", err)
	}
	return nil
}

func connectToDB(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.G.Error("cannot connect to database:", err)
		return client, err
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.G.Error("cannot connect to database:", err)
		return client, err
	}
	return client, nil
}

func getUserPass() (username, password string) {
	fmt.Print("Username / Email: ")
	fmt.Scanln(&username)

	fmt.Print("Password: ")
	b, _ := terminal.ReadPassword(int(syscall.Stdin))
	password = string(b)

	return username, password
}

func authPixiv(api *pixiv.AppAPI, c *config.Config) error {
	if c.Pixiv.RefreshToken == "" {
		u, p := getUserPass()
		api.SetUser(u, p)
		_, err := api.ForceAuth()
		if err != nil {
			return err
		}
	} else {
		api.SetRefreshToken(c.Pixiv.RefreshToken)
		_, err := api.ForceAuth()
		if pixiv.IsInvalidCredentials(err) {
			u, p := getUserPass()
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

func downloadIllusts(il []pixiv.Illust, dl *downloader.Downloader) error {
	for _, i := range il {
		if i.MetaSinglePage.OriginalImageURL != "" {
			req, err := http.NewRequest("GET", i.MetaSinglePage.OriginalImageURL, nil)
			if err != nil {
				return err
			}
			req.Header["Referer"] = []string{"https://app-api.pixiv.net"}
			dl.Add(&downloader.Task{Request: req, LocalPath: ""})
		} else {
			for _, iu := range i.MetaPages {
				req, err := http.NewRequest("GET", iu.ImageURLs.Original, nil)
				if err != nil {
					return err
				}
				dl.Add(
					&downloader.Task{Request: req, LocalPath: ""},
				)
			}
		}
	}
	return nil
}
