package cli

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/WOo0W/bowerbird/model"
	"github.com/WOo0W/bowerbird/server"

	"github.com/WOo0W/bowerbird/downloader"
	"github.com/WOo0W/bowerbird/helper"
	"github.com/WOo0W/go-pixiv/pixiv"
	"github.com/hashicorp/go-retryablehttp"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/WOo0W/bowerbird/cli/log"
	pixivh "github.com/WOo0W/bowerbird/helper/pixiv"

	"github.com/WOo0W/bowerbird/config"
	"github.com/urfave/cli/v2"
)

//New build a new bowerbird cli app
func New() *cli.App {
	conf := config.New()
	configFile := ""
	dbOnly := false

	var (
		dbc *mongo.Client
		db  *mongo.Database
	)

	var (
		pixivapi *pixiv.AppAPI
		pixivrhc *retryablehttp.Client
		pixivdl  *downloader.Downloader
	)

	initPixiv := func() error {
		pixivrhc = retryablehttp.NewClient()
		pixivrhc.Backoff = helper.DefaultBackoff
		pixivrhc.Logger = nil
		pixivrhc.RequestLogHook = func(l retryablehttp.Logger, req *http.Request, tries int) {
			log.G.Debug(fmt.Sprintf("pixiv http: %s %s tries: %d", req.Method, req.URL, tries))
		}
		tra := pixivrhc.HTTPClient.Transport.(*http.Transport)
		err := helper.SetTransportProxy(tra, conf.Pixiv.APIProxy, conf.Network.GlobalProxy)
		if err != nil {
			return err
		}
		pixivapi = pixiv.NewWithClient(pixivrhc.StandardClient())
		pixivapi.SetLanguage(conf.Pixiv.Language)

		trd := &http.Transport{}
		err = helper.SetTransportProxy(trd, conf.Pixiv.DownloaderProxy, conf.Network.GlobalProxy)
		if err != nil {
			return err
		}
		pixivdl = downloader.NewWithCliet(&http.Client{Transport: trd})

		err = authPixiv(pixivapi, conf)
		if err != nil {
			return fmt.Errorf("pixiv: auth failed: %w", err)
		}
		log.G.Info(fmt.Sprintf("pixiv: logged as %s (%d)", pixivapi.AuthResponse.Response.User.Name, pixivapi.UserID))
		conf.Pixiv.RefreshToken = pixivapi.RefreshToken
		err = conf.Save()
		if err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}
		return nil
	}

	return &cli.App{
		Name:    "Bowerbird",
		Usage:   "A toolset to manage your collection",
		Version: config.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "The path of JSON config file",
				Destination: &configFile,
			},
			&cli.BoolFlag{
				Name:        "db-only",
				Aliases:     []string{"d"},
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

			log.G.ConsoleLevel = log.ParseLevel(conf.Log.ConsoleLevel)
			log.G.FileLevel = log.ParseLevel(conf.Log.FileLevel)

			if conf.Database.Enabled {
				ctx := context.Background()
				var err error
				dbc, err = connectToDB(ctx, conf.Database.MongoURI)
				if err != nil {
					log.G.Error("cannot connect to database:", err)
					return nil
				}
				db = dbc.Database(conf.Database.DatabaseName)
				err = model.EnsureIndexes(ctx, db)
				if err != nil {
					log.G.Error(err)
					return nil
				}
			}
			return nil
		},
		Commands: []*cli.Command{
			{
				Name: "serve",
				Action: func(c *cli.Context) error {
					err := server.Serve(conf, db)
					if err != nil {
						log.G.Error(err)
					}
					return nil
				},
			},
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
						Usage:   "Specify the pixiv user ID for the operations. Default use the logged user's ID.",
					},
					&cli.IntFlag{
						Name:    "limit",
						Aliases: []string{"l"},
						Usage:   "Limit how many items to download",
					},
				},
				Before: func(c *cli.Context) error {
					err := initPixiv()
					if err != nil {
						log.G.Error(err)
						return cli.Exit("", 1)
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
								Usage: "Update all users in database despite modified date",
							},
							&cli.DurationFlag{
								Name:  "before",
								Usage: "Update users profile which was updated before the duration till now. Default: 240h",
							},
						},
						Action: func(c *cli.Context) error {
							if !conf.Database.Enabled {
								log.G.Error("User profiles are not saved without database.")
								return nil
							}
							var du time.Duration
							if c.IsSet("before") {
								du = c.Duration("before")
							} else {
								du = 240 * time.Hour
							}
							err := pixivh.UpdateAllUsers(db, pixivapi, c.Bool("all"), du)
							if err != nil {
								log.G.Error(err)
							}
							return nil
						},
					},
					{
						Name:  "illust",
						Usage: "Save illusts, manga and ugoira from pixiv",
						Subcommands: []*cli.Command{
							{
								Name:  "bookmarks",
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

									uid := getPixivUserFlag(c, pixivapi.UserID)

									var opt *pixiv.BookmarkQuery
									if c.IsSet("max-bookmark-id") {
										opt = &pixiv.BookmarkQuery{
											MaxBookmarkID: c.Int("max-bookmark-id"),
										}
									}

									r, err := pixivapi.User.BookmarkedIllusts(uid, restrict, opt)
									if err != nil {
										log.G.Error(err)
										return nil
									}

									pixivdl.Start()
									pixivh.ProcessIllusts(r, c.Int("limit"), pixivdl, pixivapi, conf.Storage.ParsedPixiv(), c.StringSlice("tags"), c.Bool("tags-match-all"), db, dbOnly)
									downloaderUILoop(pixivdl)
									return nil
								},
							},
							{
								Name:  "uploads",
								Usage: "Get user's uploaded works",
								Action: func(c *cli.Context) error {
									uid := getPixivUserFlag(c, pixivapi.UserID)

									var opt *pixiv.IllustQuery
									offset := c.Int("offset")
									if offset > 0 {
										opt = &pixiv.IllustQuery{
											Offset: offset,
										}
									}
									ri, err := pixivapi.User.Illusts(uid, opt)
									if err != nil {
										log.G.Error(err)
										return nil
									}

									pixivdl.Start()
									pixivh.ProcessIllusts(ri, c.Int("limit"), pixivdl, pixivapi, conf.Storage.ParsedPixiv(), c.StringSlice("tags"), c.Bool("tags-match-all"), db, dbOnly)
									downloaderUILoop(pixivdl)
									return nil
								},
							},
						},
					},
					{
						Name:  "novel",
						Usage: "Save novel to database from pixiv",
						Before: func(c *cli.Context) error {
							if db == nil {
								log.G.Error("Can only save novel while the database is enabled")
								return cli.Exit("", 1)
							}
							return nil
						},
						Subcommands: []*cli.Command{
							{
								Name: "bookmarks",
								Flags: []cli.Flag{
									// &cli.BoolFlag{
									// 	Name:  "save-series",
									// 	Usage: "Save full series of each bookmarked novel.",
									// },
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

									uid := getPixivUserFlag(c, pixivapi.UserID)

									var opt *pixiv.BookmarkQuery
									if c.IsSet("max-bookmark-id") {
										opt = &pixiv.BookmarkQuery{
											MaxBookmarkID: c.Int("max-bookmark-id"),
										}
									}

									rn, err := pixivapi.User.BookmarkedNovels(uid, restrict, opt)
									if err != nil {
										log.G.Error(err)
										return nil
									}
									pixivh.ProcessNovels(rn, c.Int("limit"), pixivapi, c.StringSlice("tags"), c.Bool("tags-match-all"), db)
									return nil
								},
							},
							{
								Name: "uploads",
								Action: func(c *cli.Context) error {
									uid := getPixivUserFlag(c, pixivapi.UserID)
									rn, err := pixivapi.User.Novels(uid)
									if err != nil {
										log.G.Error(err)
										return nil
									}
									pixivh.ProcessNovels(rn, c.Int("limit"), pixivapi, c.StringSlice("tags"), c.Bool("tags-match-all"), db)
									return nil
								},
							},
						},
					},
				},
			},
		},
	}
}
