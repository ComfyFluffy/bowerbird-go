package cli

import (
	"fmt"
	"net/http"

	"github.com/WOo0W/bowerbird/downloader"
	"github.com/WOo0W/bowerbird/helper"
	"github.com/WOo0W/go-pixiv/pixiv"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/WOo0W/bowerbird/cli/log"

	"github.com/WOo0W/bowerbird/config"
	"github.com/urfave/cli/v2"
)

//New returns an APP
func New() *cli.App {
	conf := config.New()
	configFile := ""
	noDB := false
	limit := 0

	var pixivapi *pixiv.AppAPI
	var pixivrhc *retryablehttp.Client
	var pixivdl *downloader.Downloader

	return &cli.App{
		Name:  "Bowerbird",
		Usage: "A toolset to manage your collection",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "The path of JSON config file",
				Destination: &configFile,
			},
			&cli.BoolFlag{
				Name:        "no-db",
				Usage:       "Do not connect to the database",
				Destination: &noDB,
			},
			&cli.IntFlag{
				Name:        "limit",
				Aliases:     []string{"l"},
				Usage:       "Limit how many items to download",
				Destination: &limit,
			},
			&cli.IntFlag{
				Name:        "offset",
				Usage:       "Start downloading with first offset items jumpped",
				Destination: &limit,
			},
		},
		Before: func(c *cli.Context) error {
			err := loadConfigFile(conf, configFile)
			if err != nil {
				log.G.Error("loading config:", err)
				return cli.Exit("", 1)
			}
			log.G.ConsoleLevel = log.SwitchLevel(conf.Log.ConsoleLevel)
			log.G.FileLevel = log.SwitchLevel(conf.Log.FileLevel)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "pixiv",
				Usage: "Get works from pixiv.net",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "tags",
						Aliases: []string{"t"},
						Usage:   "Get items with given tags",
					},
					&cli.IntFlag{
						Name:    "user",
						Aliases: []string{"u"},
						Usage:   "Specify the pixiv user id for the operations. 0 means the logged user.",
					},
				},
				Before: func(c *cli.Context) error {
					pixivrhc = retryablehttp.NewClient()
					pixivrhc.Backoff = helper.DefaultBackoff
					pixivrhc.Logger = nil
					pixivrhc.RequestLogHook = func(l retryablehttp.Logger, req *http.Request, tries int) {
						log.G.Debug(fmt.Sprintf("pixiv http: %s %s tries: %d", req.Method, req.URL, tries))
					}
					tr := pixivrhc.HTTPClient.Transport.(*http.Transport)
					if conf.Pixiv.APIProxy != "" {
						setProxy(tr, conf.Pixiv.APIProxy)
					} else if conf.Network.GlobalProxy != "" {
						setProxy(tr, conf.Network.GlobalProxy)
					}
					pixivapi = pixiv.NewWithClient(pixivrhc.StandardClient())
					pixivapi.SetLanguage(conf.Pixiv.Language)

					trd := &http.Transport{}
					if conf.Pixiv.DownloaderProxy != "" {
						setProxy(trd, conf.Pixiv.DownloaderProxy)
					} else if conf.Network.GlobalProxy != "" {
						setProxy(trd, conf.Network.GlobalProxy)
					}
					pixivdl = downloader.NewWithCliet(&http.Client{Transport: trd})

					err := authPixiv(pixivapi, conf)
					if err != nil {
						log.G.Error("pixiv: auth failed:", err)
						return cli.Exit("", 1)
					}
					log.G.Info(fmt.Sprintf("pixiv: logged as %s (%d)", pixivapi.AuthResponse.Response.User.Name, pixivapi.UserID))
					conf.Pixiv.RefreshToken = pixivapi.RefreshToken
					err = conf.Save()
					if err != nil {
						log.G.Error("saving config:", err)
					}
					return nil
				},
				Subcommands: []*cli.Command{
					{
						Name:  "bookmark",
						Usage: "Get user's bookmarked works",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "private",
								Usage: "Download the private bookmarks only",
							},
						},

						Action: func(c *cli.Context) error {
							var restrict pixiv.Restrict
							if c.Bool("private") {
								restrict = pixiv.RPrivate
							} else {
								restrict = pixiv.RPublic
							}

							var uid int
							if c.IsSet("user") {
								uid = c.Int("user")
							} else {
								uid = pixivapi.UserID
							}

							r, err := pixivapi.User.BookmarkedIllusts(uid, restrict, nil)
							if err != nil {
								log.G.Error(err)
								return cli.Exit("", 1)
							}

							pixivdl.Start()
							downloadIllusts(r, limit, pixivdl, pixivapi, conf.Storage.ParsedPixiv(), c.StringSlice("tags"))
							downloaderUILoop(pixivdl)
							return nil
						},
					},
					{
						Name:  "uploads",
						Usage: "Get user's uploaded works",
						Action: func(c *cli.Context) error {
							uid := pixivapi.UserID
							id := c.Int("user")
							if id != 0 {
								uid = id
							}

							ri, err := pixivapi.User.Illusts(uid, nil)
							if err != nil {
								return cli.Exit("", 1)
							}

							pixivdl.Start()
							downloadIllusts(ri, limit, pixivdl, pixivapi, conf.Storage.ParsedPixiv(), c.StringSlice("tags"))
							downloaderUILoop(pixivdl)
							return nil
						},
					},
				},
			},
		},
	}
}
