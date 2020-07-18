package cli

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/WOo0W/bowerbird/cli/color"
	"github.com/WOo0W/bowerbird/downloader"
	"github.com/WOo0W/bowerbird/helper"
	"github.com/WOo0W/go-pixiv/pixiv"

	"github.com/WOo0W/bowerbird/cli/log"

	"github.com/WOo0W/bowerbird/config"
	"github.com/urfave/cli/v2"
)

func New() *cli.App {
	conf := config.New()
	configFile := ""
	noDB := false

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
			&cli.BoolFlag{
				Name:        "no-db",
				Usage:       "Do not connect to the database",
				Destination: &noDB,
			},
		},
		// Load and save config file
		Before: func(c *cli.Context) error {
			err := loadConfigFile(conf, configFile)

			if err != nil {
				log.G.Error("Error loading config:", err)
				os.Exit(1)
			}
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "pixiv",
				Usage: "Get images and infomation from pixiv.net",
				Subcommands: []*cli.Command{
					{
						Name: "bookmark",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:    "user",
								Aliases: []string{"u"},
								Usage:   "Specify the pixiv user id",
							},
							&cli.BoolFlag{
								Name:  "private",
								Usage: "Download the private bookmarks only",
							},
						},
						Action: func(c *cli.Context) error {
							log.G.Info("bookmark")

							// ctx := context.Background()
							// var client *mongo.Client
							// if !noDB {
							// 	var err error
							// 	client, err = connectToDB(ctx, conf.Database.MongoURI)
							// 	if err != nil {
							// 		return nil
							// 	}
							// }

							tr := &http.Transport{}
							hc := &http.Client{Transport: log.NewLoggingRoundTripper(log.G, tr)}
							if conf.Network.Proxy != "" {
								err := setProxy(tr, conf.Network.Proxy)
								if err != nil {
									log.G.Error(err)
									return nil
								}
							}
							papi := pixiv.NewWithClient(hc)
							err := authPixiv(papi, conf)
							if err != nil {
								log.G.Error("pixiv auth failed:", err)
								return nil
							}
							restrict := pixiv.RPublic
							if c.Bool("private") {
								restrict = pixiv.RPrivate
							}
							uid := papi.UserID
							if c.IsSet("user") {
								uid = c.Int("user")
							}
							log.G.Info(fmt.Sprintf("pixiv: Logged as %s (%d)", papi.AuthResponse.Response.User.Name, papi.UserID))

							dl := downloader.New()
							dl.Start(5)

							re := &helper.Retryer{WaitMax: 10 * time.Second, WaitMin: 2 * time.Second, TriesMax: 3}

							il := []pixiv.Illust{}
							var r *pixiv.RespIllusts
							re.Retry(
								func() (err error) {
									r, err = papi.User.BookmarkedIllusts(uid, restrict, nil)
									if err != nil {
										return err
									}
									il = r.Illusts
									return nil
								},
							)

							re.Retry(
								func() (err error) {
									r, err = r.NextIllusts()
									if err != nil {
										return err
									}
									il = append(il, r.Illusts...)
									return nil
								},
							)

							// tries := 0
							// for r.NextURL != "" {
							// 	tries++
							// 	r, err = r.NextIllusts()
							// 	if err != nil {
							// 		return nil, err
							// 	}
							// }

							if err != nil {
								log.G.Error(err)
								return nil
							}

							log.G.Info(r.Illusts[:3])
							return nil
						},
					},
				},
				// Action: func(c *cli.Context) error {
				// 	log.G.Info("pixiv")
				// 	return nil
				// 	ctx := context.Background()
				// 	client, err := mogongo.Connect(ctx, options.Client().ApplyURI(conf.Database.MongoURI))
				// 	if err != nil {
				// 		log.G.Error(err)
				// 		return nil
				// 	}
				// 	defer client.Disconnect(ctx)
				// 	err = client.Ping(ctx, readpref.Primary())
				// 	if err != nil {
				// 		log.G.Error(err)
				// 		return nil
				// 	}

				// 	db := client.Database(conf.Database.DatabaseName)
				// 	mu := db.Collection(m.CollectionUser)

				// 	// pretty.Log(mu.InsertOne(ctx, m.User{Source: "test", SourceID: "123"}))

				// 	api := pixiv.New()
				// 	api.SetRefreshToken("")
				// 	api.SetProxy("http://127.0.0.1:8888")
				// 	ra, err := api.ForceAuth()
				// 	if err != nil {
				// 		log.G.Error(err)
				// 		return nil
				// 	}
				// 	fmt.Println(ra.Response.RefreshToken)

				// 	id, _ := strconv.Atoi(ra.Response.User.ID)
				// 	api.User.BookmarkedIllusts(id, "w", nil)

				// 	writeIllusts := func(ri *pixiv.RespIllusts) error {
				// 		for _, x := range ri.Illusts {
				// 			uid := strconv.Itoa(x.User.ID)
				// 			u := m.User{}
				// 			err := mu.FindOne(ctx,
				// 				b.D{{"sourceID", uid}, {"source", "pixiv"}},
				// 			).Decode(&u)

				// 			if err != nil {
				// 				if err == mongo.ErrNoDocuments {
				// 					u.Source = "pixiv"
				// 					u.SourceID = uid
				// 					u.Extension.Pixiv = &m.PixivUser{
				// 						IsFollowed: x.User.IsFollowed,
				// 					}
				// 					mu.InsertOne(ctx, u)
				// 				} else {
				// 					return err
				// 				}
				// 			} else {
				// 				if u.Extension.Pixiv == nil {
				// 					mu.UpdateOne(ctx,
				// 						b.D{{"sourceID", uid}, {"source", "pixiv"}},
				// 						b.D{{"$set", b.D{{"extension",
				// 							m.ExtUser{Pixiv: &m.PixivUser{IsFollowed: x.User.IsFollowed}}}}}},
				// 					)
				// 				}
				// 				if u.Extension.Pixiv.IsFollowed != x.User.IsFollowed {
				// 					mu.UpdateOne(ctx,
				// 						b.D{{"sourceID", uid}, {"source", "pixiv"}},
				// 						b.D{{"$set", b.D{{"isFollowed", x.User.IsFollowed}}}},
				// 					)
				// 				}
				// 			}
				// 			pretty.Log(u)
				// 		}
				// 		return nil
				// 	}

				// 	ri, err := api.User.BookmarkedIllusts(id, pixiv.RPublic, nil)
				// 	if err != nil {
				// 		log.G.Error(err)
				// 		return nil
				// 	}
				// 	writeIllusts(ri)
				// 	os.Exit(0)

				// 	for i := 0; i < 3; i++ {
				// 		ri, err = ri.NextIllusts()
				// 		if err != nil {
				// 			log.G.Error(err)
				// 			return nil
				// 		}
				// 	}

				// 	return nil
				// },
			},
		},
	}
}
