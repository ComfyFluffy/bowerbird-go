package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

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

var pximgDate = regexp.MustCompile(
	`(\d{4}/\d{2}/\d{2}/\d{2}/\d{2}/\d{2})`,
)

func setProxy(tr *http.Transport, uri string) {
	pr, err := url.Parse(uri)
	if err != nil {
		log.G.Error(err)
		return
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
			log.G.Error(err)
			return
		}
		tr.DialContext = spd.(proxy.ContextDialer).DialContext
	default:
		log.G.Error("set proxy: unsupported protocol")
		return
	}
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

	log.G.ConsoleLevel = log.SwitchLevel(conf.Log.ConsoleLevel)
	log.G.FileLevel = log.SwitchLevel(conf.Log.FileLevel)
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

// pximgSingleFileWithDate returns path like `C:\test\123\27427531_p0_20120522161622.png`.
// date: string like 2012/05/22/16/16/22
func pximgSingleFileWithDate(basePath string, userID int, u *url.URL) string {
	fn := filepath.Base(u.Path)
	i := strings.LastIndexByte(fn, '.')
	return filepath.Join(basePath, strconv.Itoa(userID), fn[:i]+"_"+strings.ReplaceAll(pximgDate.FindString(u.Path), "/", "")+fn[i:])
}

func downloadIllusts(il []pixiv.Illust, dl *downloader.Downloader, api *pixiv.AppAPI, basePath string) {
	time.Sleep(100 * time.Millisecond)
	for _, i := range il {
		if i.MetaSinglePage.OriginalImageURL != "" {
			req, err := http.NewRequest("GET", i.MetaSinglePage.OriginalImageURL, nil)
			if err != nil {
				log.G.Error(err)
				continue
			}
			api.SetHeaders(req)
			req.Header["Referer"] = []string{"https://app-api.pixiv.net/"}

			dl.Add(&downloader.Task{
				Request:   req,
				LocalPath: pximgSingleFileWithDate(basePath, i.User.ID, req.URL)})
		} else {
			for _, iu := range i.MetaPages {
				req, err := http.NewRequest("GET", iu.ImageURLs.Original, nil)
				if err != nil {
					log.G.Error(err)
					continue
				}
				api.SetHeaders(req)
				req.Header["Referer"] = []string{"https://app-api.pixiv.net/"}

				dl.Add(
					&downloader.Task{
						Request: req,
						LocalPath: filepath.Join(
							basePath, strconv.Itoa(i.User.ID),
							strconv.Itoa(i.ID)+"_"+
								strings.ReplaceAll(pximgDate.FindString(req.URL.Path), "/", ""),
							filepath.Base(req.URL.Path))},
				)
			}
		}
	}
}
