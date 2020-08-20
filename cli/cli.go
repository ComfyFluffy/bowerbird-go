package cli

import (
	"context"
	"fmt"
	"net/http"

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
	noDB := false
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

			log.G.ConsoleLevel = log.ParseLevel(conf.Log.ConsoleLevel)
			log.G.FileLevel = log.ParseLevel(conf.Log.FileLevel)

			if !noDB {
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
					return server.Serve(conf, db)
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
						Usage:   "Specify the pixiv user id for the operations. Default use the logged user's ID.",
					},
					&cli.IntFlag{
						Name:    "limit",
						Aliases: []string{"l"},
						Usage:   "Limit how many items to download",
					},
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
						},
						Action: func(c *cli.Context) error {
							err := initPixiv()
							if err != nil {
								log.G.Error(err)
								return nil
							}
							if noDB {
								log.G.Error("--no-db flag is true. cannot update.")
								return nil
							}
							err = pixivh.UpdateAllUsers(db, pixivapi, c.Bool("all"))
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
							err := initPixiv()
							if err != nil {
								log.G.Error(err)
								return nil
							}
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
							pixivh.ProcessIllusts(r, c.Int("limit"), pixivdl, pixivapi, conf.Storage.ParsedPixiv(), c.StringSlice("tags"), c.Bool("tags-match-all"), db, dbOnly)
							downloaderUILoop(pixivdl)
							return nil
						},
					},
					{
						Name:  "uploads",
						Usage: "Get user's uploaded works",
						Action: func(c *cli.Context) error {
							err := initPixiv()
							if err != nil {
								log.G.Error(err)
								return nil
							}
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
							pixivh.ProcessIllusts(ri, c.Int("limit"), pixivdl, pixivapi, conf.Storage.ParsedPixiv(), c.StringSlice("tags"), c.Bool("tags-match-all"), db, dbOnly)
							downloaderUILoop(pixivdl)
							return nil
						},
					},
				},
			},
		},
	}
}
