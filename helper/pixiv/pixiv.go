package pixiv

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/WOo0W/bowerbird/cli/log"
	"github.com/WOo0W/bowerbird/downloader"
	"github.com/WOo0W/bowerbird/model"
	"github.com/WOo0W/go-pixiv/pixiv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// regexp matchers for url of i.pximg.net
var (
	PximgDate = regexp.MustCompile(
		// 2018/11/06/00/25/50
		`(\d{4}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d{2})`,
	)
	PximgIllust = regexp.MustCompile(
		// https://i.pximg.net/img-original/img/2020/06/04/11/26/29/82078769_p0.jpg
		// https://i.pximg.net/img-original/img/2018/11/06/00/25/50/71525726_ugoira0.jpg
		`^https:\/\/i\.pximg\.net\/img-original\/img\/\d{4}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d+_(?:ugoira|p)\d+\..+$`,
	)
	PximgAvatar = regexp.MustCompile(
		// https://i.pximg.net/user-profile/img/2020/08/04/11/43/18/19112778_c80cc80ba5399b9181d26f48b222b204_170.jpg
		`^https:\/\/i\.pximg\.net\/user-profile\/img\/\d{4}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d+_[0-9a-f]{32}_170\..+$`,
	)
	PximgProfileBackground = regexp.MustCompile(
		// https://i.pximg.net/c/1200x600_90_a2_g5/background/img/2020/06/06/13/27/08/3025732_8257093997155eda44f50624057218be_master1200.jpg
		// https://i.pximg.net/c/1200x1200_90_a2_g5/background/img/2016/05/17/13/15/02/1278271_5a41eb54a8ede94a257321d1e100f739.jpg
		`^https:\/\/i\.pximg\.net\/c\/\d+x\d+_90_a2_g5\/background\/img\/\d{4}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d+_[0-9a-f]{32}.*\..+$`,
	)
	PximgWorkspaceImage = regexp.MustCompile(
		`^https:\/\/i\.pximg\.net\/workspace\/img\/\d{4}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d+_[0-9a-f]{32}\..+$`,
	)
	PximgUgoiraZip = regexp.MustCompile(
		`^https:\/\/i\.pximg\.net\/img-zip-ugoira\/img\/\d{4}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d{2}\/\d+_ugoira\d+x\d+\..+$`,
	)
)

// pximgSingleFileWithDate returns path like `123/27427531_p0_20120522161622.png`.
// date: string like 2012/05/22/16/16/22
func pximgSingleFileWithDate(userID int, u *url.URL) string {
	fn := filepath.Base(u.Path)
	i := strings.LastIndexByte(fn, '.')
	return strconv.Itoa(userID) +
		"/" + fn[:i] + "_" + strings.ReplaceAll(PximgDate.FindString(u.Path), "/", "") + fn[i:]
}

func setAfterFinishedFunc(cm *mongo.Collection, t *downloader.Task, u, fp string) {
	t.AfterFinished = func(*downloader.Task) {
		_, err := cm.UpdateOne(context.Background(),
			bson.D{{Key: "url", Value: u}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "path", Value: fp}}}})
		if err != nil {
			log.G.Error(err)
		}
	}
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

func updateUsers(db *mongo.Database, api *pixiv.AppAPI, usersToUpdate []int) {
	log.G.Info("updating", len(usersToUpdate), "user profiles...")
	for i, id := range usersToUpdate {
		// Current:
		r, err := api.User.Detail(id, nil)
		if err != nil {
			log.G.Error(err)
			continue
			// if rerr, ok := err.(*pixiv.ErrAppAPI); ok && rerr.Response.StatusCode == 403 {
			// 	log.G.Warn("got http 403: sleeping for 300s")
			// 	time.Sleep(300 * time.Second)
			// 	goto Current
			// } else {
			// 	continue
			// }
		}
		err = saveUserProfile(r, db)
		if err != nil {
			log.G.Error(err)
			continue
		}
		log.G.Info(fmt.Sprintf("[%d/%d] updated user %s (%d)", i+1, len(usersToUpdate), r.User.Name, r.User.ID))
	}
}

// ProcessIllusts processes the pixiv illusts until
// the NextURL is empty or the limit reached
func ProcessIllusts(ri *pixiv.RespIllusts, limit int, dl *downloader.Downloader, api *pixiv.AppAPI, basePath string, tags []string, tagsMatchAll bool, db *mongo.Database, dbOnly bool) {
	i := 0
	idb := 0
	usersToUpdate := make(map[int]struct{}, 120)
	var cm *mongo.Collection
	if db != nil {
		cm = db.Collection(model.CollectionMedia)
	}
Loop:
	for {
		if db != nil {
			err := saveIllusts(ri.Illusts, db, usersToUpdate)
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

				if !il.Visible {
					continue
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
					fp := pximgSingleFileWithDate(il.User.ID, req.URL)
					t := &downloader.Task{
						Request: req,
						// string like `C:\test\12345\67891_p0_20200202123456.jpg`
						LocalPath: filepath.Join(basePath, fp),
					}
					setAfterFinishedFunc(cm, t, il.MetaSinglePage.OriginalImageURL, fp)

					dl.Add(t)
				} else {
					for _, iu := range il.MetaPages {
						req, err := api.NewPximgRequest("GET", iu.ImageURLs.Original, nil)
						if err != nil {
							log.G.Error(err)
							continue
						}

						fp := strconv.Itoa(il.User.ID) + "/" +
							strconv.Itoa(il.ID) + "_" +
							strings.ReplaceAll(PximgDate.FindString(req.URL.Path), "/", "") + "/" +
							filepath.Base(req.URL.Path)

						t := &downloader.Task{
							Request: req,
							// string like `C:\test\12345\67890_2020134554\67890_p0.jpg`
							LocalPath: filepath.Join(
								basePath, fp)}
						setAfterFinishedFunc(cm, t, iu.ImageURLs.Original, fp)

						dl.Add(t)

					}
				}
				i++
			}
			log.G.Info(i, "items were sent to download queue")
		} else {
			log.G.Info(idb, "items processed to database")
			if limit != 0 && idb >= limit {
				break Loop
			}
		}

		if ri.NextURL == "" || limit != 0 && i >= limit {
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

	if len(usersToUpdate) > 0 {
		userIDs := make([]int, 0, len(usersToUpdate))
		for i := range usersToUpdate {
			userIDs = append(userIDs, i)
		}
		sort.Ints(userIDs)

		updateUsers(db, api, userIDs)
	}
}