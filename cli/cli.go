package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/WOo0W/bowerbird/cli/color"
	m "github.com/WOo0W/bowerbird/model"
	"github.com/WOo0W/go-pixiv/pixiv"
	"github.com/kr/pretty"
	b "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/net/context"

	"github.com/WOo0W/bowerbird/cli/log"

	"github.com/WOo0W/bowerbird/config"
	"github.com/cavaliercoder/grab"
	"github.com/urfave/cli/v2"
)

func loadConfigFile(conf *config.Config, path string) error {
	if path == "" {
		if err := os.MkdirAll(conf.Storage.RootDir, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		path = filepath.Join(conf.Storage.RootDir, "config.json")
		// log.G.Info("--config not set, use default file:", path)
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
				log.G.Error("Error loading config:", err)
				os.Exit(1)
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
			{
				Name: "pixiv",
				Action: func(c *cli.Context) error {
					ctx := context.Background()
					client, err := mongo.Connect(ctx, options.Client().ApplyURI(conf.Database.MongoURI))
					if err != nil {
						log.G.Error(err)
						return nil
					}
					defer client.Disconnect(ctx)
					err = client.Ping(ctx, readpref.Primary())
					if err != nil {
						log.G.Error(err)
						return nil
					}

					db := client.Database(conf.Database.DatabaseName)
					mu := db.Collection(m.CollectionUser)

					// pretty.Log(mu.InsertOne(ctx, m.User{Source: "test", SourceID: "123"}))

					api := pixiv.New()
					api.SetRefreshToken(conf.Pixiv.RefreshToken)
					api.SetProxy("http://127.0.0.1:8888")
					ra, err := api.ForceAuth()
					if err != nil {
						log.G.Error(err)
						return nil
					}
					id, _ := strconv.Atoi(ra.Response.User.ID)

					writeIllusts := func(ri *pixiv.RespIllusts) error {
						for _, x := range ri.Illusts {
							uid := strconv.Itoa(x.User.ID)
							u := m.User{}
							err := mu.FindOne(ctx,
								b.D{{"sourceID", uid}, {"source", "pixiv"}},
							).Decode(&u)

							if err != nil {
								if err == mongo.ErrNoDocuments {
									u.Source = "pixiv"
									u.SourceID = uid
									u.Extension.Pixiv = &m.PixivUser{
										IsFollowed: x.User.IsFollowed,
									}
									mu.InsertOne(ctx, u)
								} else {
									return err
								}
							} else {
								if u.Extension.Pixiv == nil {
									mu.UpdateOne(ctx,
										b.D{{"sourceID", uid}, {"source", "pixiv"}},
										b.D{{"$set", b.D{{"extension",
											m.ExtUser{Pixiv: &m.PixivUser{IsFollowed: x.User.IsFollowed}}}}}},
									)
								}
								if u.Extension.Pixiv.IsFollowed != x.User.IsFollowed {
									mu.UpdateOne(ctx,
										b.D{{"sourceID", uid}, {"source", "pixiv"}},
										b.D{{"$set", b.D{{"isFollowed", x.User.IsFollowed}}}},
									)
								}
							}
							pretty.Log(u)
						}
						return nil
					}

					ri, err := api.User.BookmarkedIllusts(id, pixiv.RPublic, nil)
					if err != nil {
						log.G.Error(err)
						return nil
					}
					writeIllusts(ri)
					os.Exit(0)

					for i := 0; i < 3; i++ {
						ri, err = ri.NextIllusts()
						if err != nil {
							log.G.Error(err)
							return nil
						}
					}

					return nil
				},
			},
			{
				Name:    "download",
				Aliases: []string{"d"},
				Action: func(c *cli.Context) error {
					client := grab.NewClient()
					req, _ := grab.NewRequest(".", "http://www.jxeduyun.com/ruanyun/jiaocai/%E9%AB%98%E4%B8%AD%E6%95%B0%E5%AD%A6%20%E9%80%89%E4%BF%AE4-1%20%E5%8C%97%E5%B8%88%E5%A4%A7%E7%89%88.pdf")

					log.G.Info("Downloading", req.URL())
					resp := client.Do(req)

					t := time.NewTicker(500 * time.Millisecond)
					defer t.Stop()

				Loop:
					for {
						select {
						case <-t.C:
							log.G.Line(fmt.Sprintf("Transferred %v / %v bytes (%.2f%%)",
								resp.BytesComplete(),
								resp.Size,
								100*resp.Progress()))

						case <-resp.Done:
							break Loop
						}
					}

					// check for errors
					if err := resp.Err(); err != nil {
						log.G.Error("Download failed:", err)
						return nil
					}

					log.G.Info("Download saved to", resp.Filename)
					return nil
				},
			},
		},
	}
}
