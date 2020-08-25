package cli

import (
	"context"
	"fmt"
	"net/http"

	"github.com/WOo0W/bowerbird/downloader"
	"github.com/WOo0W/bowerbird/helper"
	"github.com/WOo0W/go-pixiv/pixiv"
	"github.com/hashicorp/go-retryablehttp"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/WOo0W/bowerbird/cli/log"

	"github.com/WOo0W/bowerbird/config"
	"github.com/urfave/cli/v2"
)

//New returns an APP
func New() *cli.App {
	conf := config.New()
	configFile := ""
	noDB := false
	dbOnly := false

	var dbc *mongo.Client
	var db *mongo.Database

	var pixivapi *pixiv.AppAPI
	var pixivrhc *retryablehttp.Client
	var pixivdl *downloader.Downloader

	return &cli.App{
		Name:    "Bowerbird",
		Usage:   "A toolset to manage your collection",
		Version: "0.0.1",
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
			&cli.BoolFlag{
				Name:        "db-only",
				Usage:       "Save items to database but not download them",
				Destination: &dbOnly,
			},
		},
		Before: func(c *cli.Context) error {
			err := loadConfigFile(conf, configFile)
			if err != nil {
				log.G.Error("loading config:", err)
				return nil
			}

			log.G.ConsoleLevel = log.SwitchLevel(conf.Log.ConsoleLevel)
			log.G.FileLevel = log.SwitchLevel(conf.Log.FileLevel)

			if !noDB {
				ctx := context.Background()
				var err error
				dbc, err = connectToDB(ctx, conf.Database.MongoURI)
				if err != nil {
					log.G.Error("cannot connect to database:", err)
					return nil
				}
				db = dbc.Database(conf.Database.DatabaseName)
				ensureIndexes(ctx, db)
			}
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
						Usage:   "Get items which have any of given tags",
					},
					&cli.BoolFlag{
						Name:  "tags-match-all",
						Usage: "Get items which have all of given tags",
					},
					&cli.IntFlag{
						Name:    "user",
						Aliases: []string{"u"},
						Usage:   "Specify the pixiv user id for the operations. Default use the logged user's ID.",
					},
					&cli.IntFlag{
						Name:    "limit",
						Aliases: []string{"l"},
						Usage:   "Limit how many items to download",
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
					pixivdl = downloader.NewWithDefaultClient()

					err := authPixiv(pixivapi, conf)
					if err != nil {
						log.G.Error("pixiv: auth failed:", err)
						return nil
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
						Name:  "update-users",
						Usage: "Update user profile in database",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "all",
								Usage: "Update all users despite modified date",
							},
						},
						Action: func(c *cli.Context) error {
							if noDB {
								log.G.Error("--no-db flag is true. cannot update.")
								return nil
							}
							err := updateAllPixivUsers(db, pixivapi, c.Bool("all"))
							if err != nil {
								log.G.Error(err)
							}
							return nil
						},
					},
					{
						Name:  "bookmark",
						Usage: "Get user's bookmarked works",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "private",
								Usage: "Download the private bookmarks only",
							},
							&cli.IntFlag{
								Name:  "max-bookmark-id",
								Usage: "Specify max_bookmark_id field",
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
							var opt *pixiv.BookmarkQuery
							maxBookmarkID := c.Int("max-bookmark-id")
							if maxBookmarkID > 0 {
								opt = &pixiv.BookmarkQuery{
									MaxBookmarkID: maxBookmarkID,
								}
							}
							r, err := pixivapi.User.BookmarkedIllusts(uid, restrict, opt)
							if err != nil {
								log.G.Error(err)
								return nil
							}

							pixivdl.Start()
							processIllusts(r, c.Int("limit"), pixivdl, pixivapi, conf.Storage.ParsedPixiv(), c.StringSlice("tags"), c.Bool("tags-match-all"), db, dbOnly)
							downloaderUILoop(pixivdl)
							return nil
						},
					},
					{
						Name:  "uploads",
						Usage: "Get user's uploaded works",
						Action: func(c *cli.Context) error {
							var uid int
							if c.IsSet("user") {
								uid = c.Int("user")
							} else {
								uid = pixivapi.UserID
							}
							var opt *pixiv.IllustQuery
							offset := c.Int("offset")
							if offset > 0 {
								opt = &pixiv.IllustQuery{
									Offset: offset,
								}
							}
							ri, err := pixivapi.User.Illusts(uid, opt)
							if err != nil {
								return nil
							}

							pixivdl.Start()
							downloaderUILoop(pixivdl)
							processIllusts(ri, c.Int("limit"), pixivdl, pixivapi, conf.Storage.ParsedPixiv(), c.StringSlice("tags"), c.Bool("tags-match-all"), db, dbOnly)
							return nil
						},
					},
				},
			},
		},
	}
}
