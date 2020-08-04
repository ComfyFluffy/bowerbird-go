package cli

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/WOo0W/bowerbird/cli/color"
	"github.com/WOo0W/bowerbird/cli/log"
	"github.com/WOo0W/bowerbird/config"
	"github.com/WOo0W/bowerbird/downloader"
	"github.com/WOo0W/go-pixiv/pixiv"
	"github.com/dustin/go-humanize"
	"go.mongodb.org/mongo-driver/mongo"
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
			log.G.Info("creating new config file:", path)
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

// pximgSingleFileWithDate returns path like `C:\test\123\27427531_p0_20120522161622.png`.
// date: string like 2012/05/22/16/16/22
func pximgSingleFileWithDate(basePath string, userID int, u *url.URL) string {
	fn := filepath.Base(u.Path)
	i := strings.LastIndexByte(fn, '.')
	return filepath.Join(basePath, strconv.Itoa(userID), fn[:i]+"_"+strings.ReplaceAll(pximgDate.FindString(u.Path), "/", "")+fn[i:])
}

//hasEveryTag checks if every tag is in the input tags
func hasEveryTag(src []pixiv.Tag, check ...string) bool {
	for _, i := range check {
		has := false
		for _, j := range src {
			if i == j.Name || i == j.TranslatedName {
				has = true
				break
			}
		}
		if !has {
			return false
		}
	}
	return true
}

//hasAnyTag checks if any tag in check matches the tags in src
func hasAnyTag(src []pixiv.Tag, check ...string) bool {
	for _, i := range src {
		for _, j := range check {
			if j == i.Name || j == i.TranslatedName {
				return true
			}
		}
	}
	return false
}

func updatePixivUsers(db *mongo.Database, api *pixiv.AppAPI, usersToUpdate []int) {
	log.G.Info("updating", len(usersToUpdate), "user profiles...")
	for i, id := range usersToUpdate {
		r, err := api.User.Detail(id, nil)
		if err != nil {
			log.G.Error(err)
			return
		}
		err = savePixivUserProfileToDB(r, db)
		if err != nil {
			log.G.Error(err)
			continue
		}
		log.G.Info(fmt.Sprintf("[%d/%d] updated user %s (%d)", i+1, len(usersToUpdate), r.User.Name, r.User.ID))
	}
}

func processIllusts(ri *pixiv.RespIllusts, limit int, dl *downloader.Downloader, api *pixiv.AppAPI, basePath string, tags []string, tagsMatchAll bool, db *mongo.Database, dbOnly bool) {
	i := 0
	idb := 0
	usersToUpdate := make(map[int]bool, 120)

Loop:
	for {
		if db != nil {
			err := savePixivIllusts(ri.Illusts, db, usersToUpdate)
			if err != nil {
				log.G.Error(err)
				return
			}
			idb += len(ri.Illusts)
		}
		if !dbOnly {
			for _, il := range ri.Illusts {
				if limit != 0 && i >= limit {
					break Loop
				}

				if len(tags) != 0 {
					if tagsMatchAll {
						if !hasEveryTag(il.Tags, tags...) {
							continue
						}
					} else {
						if !hasAnyTag(il.Tags, tags...) {
							continue
						}
					}
				}

				if il.MetaSinglePage.OriginalImageURL != "" {
					req, err := api.NewPximgRequest("GET", il.MetaSinglePage.OriginalImageURL, nil)
					if err != nil {
						log.G.Error(err)
						continue
					}

					dl.Add(&downloader.Task{
						Request: req,
						// string like `C:\test\12345\67891_p0_20200202123456.jpg`
						LocalPath: pximgSingleFileWithDate(basePath, il.User.ID, req.URL)})

				} else {
					for _, iu := range il.MetaPages {
						req, err := api.NewPximgRequest("GET", iu.ImageURLs.Original, nil)
						if err != nil {
							log.G.Error(err)
							continue
						}

						dl.Add(
							&downloader.Task{
								Request: req,
								// string like `C:\test\12345\67890_2020134554\67890_p0.jpg`
								LocalPath: filepath.Join(
									basePath, strconv.Itoa(il.User.ID),
									strconv.Itoa(il.ID)+"_"+
										strings.ReplaceAll(pximgDate.FindString(req.URL.Path), "/", ""),
									filepath.Base(req.URL.Path))},
						)
					}
				}
				i++
			}
			log.G.Info(i, "items has been sent to download queue")
		} else {
			log.G.Info(idb, "items processed to database")
			if limit != 0 && idb >= limit {
				break Loop
			}
		}

		if ri.NextURL == "" {
			break Loop
		}

		var err error
		ri, err = ri.NextIllusts()
		if err != nil {
			log.G.Error(err)
			return
		}
	}
	log.G.Info("all", i, "items processed")

	userIDs := make([]int, 0, len(usersToUpdate))
	for i := range usersToUpdate {
		userIDs = append(userIDs, i)
	}
	sort.Ints(userIDs)

	updatePixivUsers(db, api, userIDs)
}

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
