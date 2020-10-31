package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WOo0W/bowerbird/model"
	m "github.com/WOo0W/bowerbird/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
)

type D = bson.D

const dbFile = `pixivmanager.sqlite.db`
const mongoURI = "mongodb://localhost"

type Illust struct {
	ID             int
	AuthorID       int
	Date           time.Time
	IsBookmarked   bool
	TotalBookmarks int
	TotalViews     int
	Type           int
	IsVisible      bool
	Title          string
	Caption        string
	PageCount      int
	ImageURLInfo   string

	Tags []Tag `gorm:"-"`
}

type Tag struct {
	Text        string `gorm:"text"`
	Translation string `gorm:"translation"`
}

type User struct {
	ID         int
	IsFollowed bool
	Avatar,
	Name,
	Account,
	Background,
	Birth,
	Country,
	Comment string
	Gender    int
	IsPremium bool
	TotalFollowingUsers,
	TotalPublicIllustBookmarks,
	TotalIllusts,
	TotalManga,
	TotalNovels,
	TotalIllustSeries,
	TotalNovelSeries int
	TwitterAccount,
	WebPage string
}

type Ugoira struct {
	ZipURLInfo string `gorm:"column:delay_info"`
	DelayInfo  string `gorm:"column:zip_url_info"`
	Delay      []int  `gorm:"-"`
}

type MongoID struct {
	ID primitive.ObjectID `bson:"_id"`
}

type A = primitive.A

var (
	optsFOAIDOnly = options.FindOneAndUpdate().
			SetUpsert(true).SetReturnDocument(options.After).
			SetProjection(D{{"_id", 1}})
	optsUUpsert = options.Update().SetUpsert(true)
)

func buildRegexQuery(s string) primitive.Regex {
	return primitive.Regex{Options: "i", Pattern: "^" + regexp.QuoteMeta(s) + "$"}
}

func lookupObjectID(r bson.Raw) primitive.ObjectID {
	return r.Lookup("_id").ObjectID()
}

func insertMediaWithURL(ctx context.Context, cm *mongo.Collection, url string) (primitive.ObjectID, error) {
	r, err := cm.FindOneAndUpdate(ctx,
		D{{"url", url}}, A{},
		optsFOAIDOnly).DecodeBytes()
	if err != nil {
		return primitive.ObjectID{}, err
	}
	return lookupObjectID(r), nil
}

func ensureIndexes(ctx context.Context, db *mongo.Database) {
	cu := db.Collection(model.CollectionUser)
	cud := db.Collection(model.CollectionUserDetail)
	ct := db.Collection(model.CollectionTag)
	cp := db.Collection(model.CollectionPost)
	cpd := db.Collection(model.CollectionPostDetail)

	cu.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: D{{"source", 1}, {"sourceID", 1}},
			Options: options.Index().
				SetUnique(true),
		},
	)
	cud.Indexes().CreateMany(
		ctx, []mongo.IndexModel{
			{
				Keys: D{{"userID", 1}},
			},
			{
				Keys: D{{"name", 1}},
			},
		},
	)

	cp.Indexes().CreateMany(
		ctx, []mongo.IndexModel{
			{
				Keys: bson.M{"source": 1, "sourceID": 1},
				Options: options.Index().
					SetUnique(true),
			},
			{
				Keys: D{{"tagIDs", 1}},
			},
		},
	)
	cpd.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: D{{"postID", 1}},
		},
	)

	ct.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: D{{"alias", 1}, {"source", 1}},
		},
	)
}

func loadDB() {
	db, err := gorm.Open("sqlite3", dbFile)
	db.LogMode(false)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	mdb := client.Database("bowerbird")
	ensureIndexes(ctx, mdb)

	mcu := mdb.Collection(m.CollectionUser)
	mcud := mdb.Collection(m.CollectionUserDetail)
	mct := mdb.Collection(m.CollectionTag)
	mcp := mdb.Collection(m.CollectionPost)
	mcpd := mdb.Collection(m.CollectionPostDetail)
	mcm := mdb.Collection(m.CollectionMedia)

	log.Print("User start")
	t1 := time.Now()
	ru := []User{}
	db.Table("users").Find(&ru)
	for _, iu := range ru {
		u := m.User{
			Source:   "pixiv-illust",
			SourceID: strconv.Itoa(iu.ID),
			Extension: &m.ExtUser{
				Pixiv: &m.PixivUser{
					TotalFollowing:       iu.TotalFollowingUsers,
					TotalIllustSeries:    iu.TotalIllustSeries,
					TotalIllusts:         iu.TotalIllusts,
					TotalManga:           iu.TotalManga,
					TotalNovelSeries:     iu.TotalNovelSeries,
					TotalNovels:          iu.TotalNovels,
					TotalPublicBookmarks: iu.TotalPublicIllustBookmarks,
					IsFollowed:           iu.IsFollowed,
				},
			},
		}
		if iu.Avatar != "" && !strings.HasPrefix(iu.Avatar, "https://s.pximg.net") {
			//medium=https://i.pximg.net/user-profile/img/2010/03/04/22/53/18/1543667_032b291f149709c4b9c88614ef11f7f9_170.jpg
			aus := strings.Split(iu.Avatar, ";")
			aurl := fmt.Sprintf("https://i.pximg.net/user-profile/img/%s/%d_%s_170.%s", aus[0], iu.ID, aus[1], aus[2])
			id, err := insertMediaWithURL(ctx, mcm, aurl)
			if err != nil {
				log.Fatal(err)
			}
			u.AvatarIDs = []primitive.ObjectID{id}
		}
		mir, err := mcu.InsertOne(ctx, &u)
		if err != nil {
			log.Fatal(err)
		}

		var gender string
		switch iu.Gender {
		case 1:
			gender = "male"
		case 2:
			gender = "female"
		}

		ud := m.UserDetail{
			UserID: mir.InsertedID.(primitive.ObjectID),
			Name:   iu.Name,
			Extension: &m.ExtUserDetail{
				Pixiv: &m.PixivUserProfile{
					Account:        iu.Account,
					Birth:          iu.Birth,
					Region:         iu.Country,
					Gender:         gender,
					IsPremium:      iu.IsPremium,
					WebPage:        iu.WebPage,
					TwitterAccount: iu.TwitterAccount,
					Bio:            iu.Comment,
				},
			},
		}

		if iu.Background != "" && !strings.HasPrefix(iu.Background, "https://s.pximg.net") {
			sp := strings.Split(iu.Background, ";")
			id, err := insertMediaWithURL(ctx, mcm, fmt.Sprintf("https://i.pximg.net/c/1200x600_90_a2_g5/background/img/%s/%d_%s_master1200.%s", sp[0], iu.ID, sp[1], sp[2]))
			if err != nil {
				log.Fatal(err)
			}
			ud.Extension.Pixiv.BackgroundMediaID = id
		}

		_, err = mcud.InsertOne(ctx, &ud)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Print("User done ", time.Now().Sub(t1))

	t2 := time.Now()
	rt := []Tag{}
	db.Table("tags").Find(&rt)
	// mtlang := map[string]string{}
	// tss := [][]string{}
	for _, it := range rt {
		ts := make([]string, 0, 2)
		// as := []m.TagAlias{m.TagAlias{Text: it.Text}}
		if it.Text != "" {
			ts = append(ts, it.Text)
		}
		if it.Translation != "" {
			ts = append(ts, it.Translation)
			// as = append(as, m.TagAlias{Text: it.Translation, Language: "zh-cn"})
		}
		if len(ts) > 0 {
			var (
				err error
			)
			if len(ts) > 1 {
				_, err = mct.UpdateOne(ctx,
					D{{"source", m.PostSourcePixivIllust}, {"alias", D{{"$in", ts}}}},
					D{{"$addToSet", D{
						{"alias", D{
							{"$each", ts}}}}}},
					optsUUpsert)
			} else if len(ts) == 1 {
				_, err = mct.UpdateOne(ctx,
					D{{"source", m.PostSourcePixivIllust}, {"alias", ts[0]}},
					D{{"$setOnInsert", D{{"alias", ts}}}},
					optsUUpsert)
			}
			if err != nil {
				log.Fatal(err)
			}
		}
		// log.Print(time.Now().Sub(t3), ass)
	}

	log.Print("Tags done ", time.Now().Sub(t2))

	t2 = time.Now()
	mu := map[int]primitive.ObjectID{}
	tus := []m.User{}
	cur, err := mcu.Find(ctx, bson.D{}, options.Find().SetProjection(bson.M{"_id": 1, "sourceID": 1}))
	if err != nil {
		log.Fatal(err)
	}
	cur.All(ctx, &tus)
	for _, u := range tus {
		ids, _ := strconv.Atoi(u.SourceID)
		mu[ids] = u.ID
	}
	log.Print("find user id done ", time.Now().Sub(t2))

	t2 = time.Now()
	ois := []Illust{}
	db.Table("illusts").Find(&ois)
	for _, il := range ois {
		ts := []Tag{}
		db.Table("tags").Joins("JOIN illusts_tags ON illusts_tags.tag_id = tags.id AND illusts_tags.illust_id = ?", il.ID).Find(&ts)
		sts := []string{}
		for _, t := range ts {
			if t.Text != "" {
				sts = append(sts, t.Text)
			}
			if t.Translation != "" {
				sts = append(sts, t.Translation)
			}
		}

		rtids := []primitive.ObjectID{}
		if len(sts) > 0 {
			rts := []m.Tag{}
			cur, err := mct.Find(ctx, bson.M{"alias": bson.M{"$in": sts}})
			if err != nil {
				log.Fatal(err)
			}
			cur.All(ctx, &rts)
			for _, rt := range rts {
				rtids = append(rtids, rt.ID)
			}
		}

		oid, ok := mu[il.AuthorID]
		if !ok {
			log.Fatal("aid", il)
		}
		ri := m.Post{
			OwnerID:         oid,
			Source:          "pixiv-illust",
			SourceID:        strconv.Itoa(il.ID),
			SourceInvisible: !il.IsVisible,
			Extension: &m.ExtPost{
				Pixiv: &m.PixivIllust{
					TotalBookmarks: il.TotalBookmarks,
					IsBookmarked:   il.IsBookmarked,
					TotalViews:     il.TotalViews,
				},
			},
			TagIDs: rtids,
		}
		mir, err := mcp.InsertOne(ctx, &ri)
		if err != nil {
			log.Fatal(err)
		}

		iugd := Ugoira{}
		var it string
		switch il.Type {
		case 1:
			it = "illust"
		case 2:
			it = "manga"
		case 3:
			it = "ugoira"
			db.Table("ugoiras").First(&iugd, "illust_id = ?", il.ID)
			// log.Fatal("QAQ")
			if iugd.DelayInfo != "" {
				for _, s := range strings.Split(iugd.DelayInfo, ";") {
					si, _ := strconv.Atoi(s)
					iugd.Delay = append(iugd.Delay, si)
				}
			}
		}

		pmids := []primitive.ObjectID{}
		if il.ImageURLInfo != "" {
			urlinfo := strings.Split(il.ImageURLInfo, ";")
			if it == "ugoira" {
				um := &m.Media{
					URL: fmt.Sprintf("https://i.pximg.net/img-original/img/%s/%d_p%d.%s", urlinfo[0], il.ID, 0, urlinfo[1]),
				}
				inr, err := mcm.InsertOne(ctx, um)
				if err != nil {
					log.Fatal(err)
				}
				pmids = append(pmids, inr.InsertedID.(primitive.ObjectID))

				um = &m.Media{
					Extension: &m.ExtMedia{
						Pixiv: &m.PixivMedia{
							UgoiraDelay: iugd.Delay,
						},
					},
					URL: fmt.Sprintf("https://i.pximg.net/img-zip-ugoira/img/%s/%d_ugoira600x600.zip", iugd.ZipURLInfo, il.ID),
				}
				inr, err = mcm.InsertOne(ctx, um)
				if err != nil {
					log.Fatal(err)
				}
				pmids = append(pmids, inr.InsertedID.(primitive.ObjectID))
			} else {
				if len(urlinfo) == 2 {
					for i := 0; i < il.PageCount; i++ {
						var u string
						u = fmt.Sprintf("https://i.pximg.net/img-original/img/%s/%d_p%d.%s", urlinfo[0], il.ID, i, urlinfo[1])

						inr, err := mcm.InsertOne(ctx, &m.Media{
							URL: u,
							// Path: fmt.Sprintf("pixiv/%d/%d-p%d.", il.AuthorID, il.ID,i,),
						})
						if err != nil {
							log.Fatal(err)
						}
						pmids = append(pmids, inr.InsertedID.(primitive.ObjectID))
					}
				}
			}
		}

		pd := m.PostDetail{
			PostID:   mir.InsertedID.(primitive.ObjectID),
			Date:     il.Date,
			MediaIDs: pmids,
			Extension: &m.ExtPostDetail{
				Pixiv: &m.PixivIllustDetail{
					Type:        it,
					CaptionHTML: il.Caption,
					Title:       il.Title,
				},
			},
		}
		if il.PageCount > 70 {
			log.Print(pd, il)
		}
		mcpd.InsertOne(ctx, &pd)
	}
	log.Print("Illusts done ", time.Now().Sub(t2))

}

func main() {
	// cli.New().Run(os.Args)
	// loadDB()
	// chTags()
	// testTags()
	// testWebDav()
	// testCaption()
	loadDB()
}
