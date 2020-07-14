package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/webdav"

	"golang.org/x/net/html/atom"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"

	m "github.com/WOo0W/bowerbird/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jinzhu/gorm"

	aop "github.com/adam-hanna/arrayOperations"
	_ "github.com/mattn/go-sqlite3"
)

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
	DelayInfo string `gorm:"column:zip_url_info"`
	Delay     []int  `gorm:"-"`
}

type MongoID struct {
	ID primitive.ObjectID `bson:"_id"`
}

func ensureIndexes(ctx context.Context, mdb *mongo.Database) {
	mcu := mdb.Collection(m.CollectionUser)
	mcud := mdb.Collection(m.CollectionUserDetail)
	mct := mdb.Collection(m.CollectionTag)
	mcp := mdb.Collection(m.CollectionPost)
	mcpd := mdb.Collection(m.CollectionPostDetail)

	mcu.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{"source": 1, "sourceID": 1},
			Options: options.Index().
				SetUnique(true),
		},
	)
	mcud.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{"userID": 1},
		},
	)

	mcp.Indexes().CreateMany(
		ctx, []mongo.IndexModel{
			{
				Keys: bson.M{"source": 1, "sourceID": 1},
				Options: options.Index().
					SetUnique(true),
			},
			{
				Keys: bson.M{"tagIDs": 1},
			},
		},
	)
	mcpd.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{"postID": 1},
		},
	)

	mct.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{"alias.text": 1},
			// Options: options.Index().
			// 	SetUnique(true),
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
	mdb := client.Database("bowerbird_test")
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
			Source:   "pixiv",
			SourceID: strconv.Itoa(iu.ID),
			Extension: m.ExtUser{
				Pixiv: &m.PixivUser{
					TotalFollowing:       iu.TotalFollowingUsers,
					TotalIllustSeries:    iu.TotalIllustSeries,
					TotalIllusts:         iu.TotalIllusts,
					TotalManga:           iu.TotalManga,
					TotalNovelSeries:     iu.TotalNovelSeries,
					TotalNovels:          iu.TotalNovels,
					TotalPublicBookmarks: iu.TotalPublicIllustBookmarks,
				},
			},
		}
		mir, err := mcu.InsertOne(ctx, &u)
		if err != nil {
			if err, ok := err.(mongo.WriteException); ok {
				if len(err.WriteErrors) > 0 && err.WriteErrors[0].Code == 11000 {
					continue
				}
			}
			log.Fatal(err)
		}

		var gender string
		switch iu.Gender {
		case 1:
			gender = "m"
		case -1:
			gender = "f"
		}

		var mds string
		if iu.Comment != "" {
			// mds, err = mdConv.ConvertString(iu.Comment)
			strings.ReplaceAll(html.EscapeString(iu.Comment), "\n", "<br />")
		}
		if err != nil {
			log.Fatal(err)
		}

		ud := m.UserDetail{
			UserID: mir.InsertedID.(primitive.ObjectID),
			Name:   iu.Name,
			Extension: &m.ExtUserDetail{
				Pixiv: &m.PixivUserProfile{
					Account:        iu.Account,
					Birth:          iu.Birth,
					Country:        iu.Country,
					Gender:         gender,
					IsPremium:      iu.IsPremium,
					WebPage:        iu.WebPage,
					TwitterAccount: iu.TwitterAccount,
				},
			},
			Page: &m.Page{
				HTML: mds,
			},
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
		// if it.Text != "" && mtlang[it.Text] == "" {
		// 	mtlang[it.Text] = ""
		// }
		// if it.Translation != "" {
		// 	mtlang[it.Translation] = "zh-cn"
		// }
		// for _, v := range tss {
		// 	if tss
		// }
		// if it.Text == "" {
		// 	continue
		// }
		// t3 := time.Now()
		ass := []string{it.Text}
		// as := []m.TagAlias{m.TagAlias{Text: it.Text}}
		if it.Translation != "" {
			ass = append(ass, it.Translation)
			// as = append(as, m.TagAlias{Text: it.Translation, Language: "zh-cn"})
		}
		rts := []primitive.Regex{}
		for _, x := range ass {
			rts = append(rts, primitive.Regex{Pattern: "^" + regexp.QuoteMeta(x) + "$", Options: "i"})
		}
		r := mct.FindOne(ctx, bson.M{"alias.text": bson.M{"$in": rts}})
		err := r.Err()
		if err != nil && err != mongo.ErrNoDocuments {
			log.Fatal(err)
		}
		if err == nil {
			t := m.Tag{}
			r.Decode(&t)
			mtla := map[string]string{}
			for _, ta := range t.Alias {
				mtla[ta.Text] = ta.Language
			}
			if _, ok := mtla[it.Text]; !ok {
				mtla[it.Text] = ""
			}
			if it.Translation != "" && mtla[it.Translation] == "" {
				mtla[it.Translation] = "zh-cn"
			}
			as := []m.TagAlias{}
			for k, v := range mtla {
				as = append(as, m.TagAlias{Text: k, Language: v})
			}
			_, err := mct.UpdateOne(ctx, bson.M{"_id": t.ID}, bson.M{"$set": bson.M{"alias": as}})
			if err != nil {
				log.Fatal(err)
			}
		} else {
			t := m.Tag{Alias: []m.TagAlias{{Text: it.Text}}}
			if it.Translation != "" {
				t.Alias = append(t.Alias, m.TagAlias{Text: it.Translation, Language: "zh-cn"})
			}
			mct.InsertOne(ctx, &t)
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
		rts := []m.Tag{}

		otss := []string{}
		for _, t := range ts {
			otss = append(otss, t.Text, t.Translation)
		}
		otss = aop.DistinctString(otss)

		cur, err := mct.Find(ctx, bson.M{"alias.text": bson.M{"$in": &otss}})
		if err != nil {
			log.Fatal(err)
		}
		cur.All(ctx, &rts)
		rtids := []primitive.ObjectID{}
		for _, rt := range rts {
			rtids = append(rtids, rt.ID)
		}

		oid, ok := mu[il.AuthorID]
		if !ok {
			log.Fatal("aid", il)
		}
		ri := m.Post{
			OwnerID:       oid,
			Source:        "pixiv",
			SourceID:      strconv.Itoa(il.ID),
			SourceDeleted: !il.IsVisible,
			Extension: &m.ExtPost{
				Pixiv: &m.PixivPost{
					TotalBookmarks: il.TotalBookmarks,
					IsBookmarked:   il.IsBookmarked,
					TotalViews:     il.TotalViews,
				},
			},
			TagIDs: rtids,
		}
		mir, err := mcp.InsertOne(ctx, &ri)
		if err != nil {
			if err, ok := err.(mongo.WriteException); ok {
				if len(err.WriteErrors) > 0 && err.WriteErrors[0].Code == 11000 {
					continue
				}
			}
			log.Fatal(err)
		}

		var pmd string
		if il.Caption != "" {
			pmd = il.Caption
		}
		if il.Title != "" {
			pmd = fmt.Sprintf("<h1>%s</h1>%s", il.Title, pmd)
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

		var pmfmt string
		urlinfo := strings.Split(il.ImageURLInfo, ";")
		if il.ImageURLInfo != "" {
			switch urlinfo[1] {
			case "jpg", "jpeg":
				pmfmt = "image/jpeg"
			case "png":
				pmfmt = "image/png"
			case "gif":
				pmfmt = "image/gif"
			}
		}
		pmids := []primitive.ObjectID{}
		if it == "ugoira" {
			um := &m.Media{
				MIME: "application/zip",
				Extension: &m.ExtMedia{
					Pixiv: &m.PixivMedia{
						UgoiraDelay: iugd.Delay,
					},
				},
			}
			inr, err := mcm.InsertOne(ctx, um)
			// log.Print("ugoira", il.ID, iugd.Delay, iugd.DelayInfo)
			if err != nil {
				log.Fatal(err)
			}
			pmids = append(pmids, inr.InsertedID.(primitive.ObjectID))
		} else {
			for i := 0; i < il.PageCount; i++ {
				var u string
				if len(urlinfo) == 2 {
					u = fmt.Sprintf("https://i.pximg.net/img-original/img/%s/%d_p%d.%s", urlinfo[0], il.ID, i, urlinfo[1])
				}
				inr, err := mcm.InsertOne(ctx, &m.Media{
					MIME: pmfmt,
					URL:  u,
					// Path: fmt.Sprintf("pixiv/%d/%d-p%d.", il.AuthorID, il.ID,i,),
				})
				if err != nil {
					log.Fatal(err)
				}
				pmids = append(pmids, inr.InsertedID.(primitive.ObjectID))
			}
		}

		pd := m.PostDetail{
			PostID: mir.InsertedID.(primitive.ObjectID),
			Date:   il.Date,
			Page: &m.Page{
				HTML:     pmd,
				MediaIDs: pmids,
			},
			Extension: &m.ExtPostDetail{
				Pixiv: &m.PixivIllustDetail{
					Type: it,
				},
			},
		}
		mcpd.InsertOne(ctx, &pd)
	}
	log.Print("Illusts done ", time.Now().Sub(t2))

}

// func chTags() {
// 	ctx := context.Background()
// 	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer client.Disconnect(ctx)
// 	mdb := client.Database("bowerbird_test")
// 	ensureIndexes(ctx, mdb)
// 	mcp := mdb.Collection(m.Post{}.DBCollection())
// 	mct := mdb.Collection(m.Tag{}.DBCollection())

// 	cur, err := mct.Find(ctx, bson.M{})
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	ts := []m.Tag{}
// 	err = cur.All(ctx, &ts)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	deletedIDs := map[primitive.ObjectID]bool{}
// 	for _, t := range ts {
// 		if deletedIDs[t.ID] {
// 			continue
// 		}
// 		rts := []primitive.Regex{}
// 		for _, x := range t.Alias {
// 			rts = append(rts, primitive.Regex{Pattern: "^" + regexp.QuoteMeta(x) + "$", Options: "i"})
// 		}
// 		t1 := time.Now()
// 		cur, err := mct.Find(ctx, bson.M{"alias": bson.M{"$in": rts}})
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		ts := []m.Tag{}
// 		cur.All(ctx, &ts)
// 		tsr := t.Alias
// 		tsids := []primitive.ObjectID{}
// 		for _, t1 := range ts {
// 			if t1.ID == t.ID {
// 				continue
// 			}
// 			tsr = append(tsr, t1.Alias...)
// 			tsids = append(tsids, t1.ID)
// 		}
// 		tsr = aop.DistinctString(tsr)
// 		if len(tsr) == len(t.Alias) {
// 			// log.Print("jump", t, ts)
// 			continue
// 		}
// 		if len(tsids) > 0 {
// 			for _, x := range tsids {
// 				deletedIDs[x] = true
// 			}
// 			cur, err := mcp.Find(ctx, bson.M{"tagIDs": bson.M{"$in": tsids}}, options.Find().SetProjection(bson.M{"_id": 1}))
// 			fr := []MongoID{}
// 			cur.All(ctx, &fr)
// 			ids := []primitive.ObjectID{}
// 			for _, x := range fr {
// 				ids = append(ids, x.ID)
// 			}
// 			_, err = mcp.UpdateMany(ctx, bson.M{}, bson.M{"$pull": bson.M{"tagIDs": bson.M{"$in": tsids}}})
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 			_, err = mcp.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": ids}}, bson.M{"$addToSet": bson.M{"alias": t.ID}})
// 			if err != nil {
// 				log.Fatal(err)
// 			}

// 			_, err = mct.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": tsids}})
// 			if err != nil {
// 				log.Print(err)
// 			}
// 		}
// 		_, err = mct.UpdateOne(ctx, bson.M{"_id": t.ID}, bson.M{"$set": bson.M{"alias": tsr}})
// 		if err != nil {
// 			log.Println(time.Now().Sub(t1), ts, tsr, tsids)
// 			log.Fatal(err)
// 		}
// 		log.Println(time.Now().Sub(t1), t, ts, tsr, tsids)
// 	}
// }

func testCaption() {
	s := `
宣伝）FPRM|あいかわらず今月と来月のリワード、$10以上でーす<br />いまは新刊途中<br /><strong><a href="pixiv://illusts/74660277">illust/74660277</a></strong><br /><br />MyWebsite：<a href="http://waysin.net/" target="_blank">http://waysin.net/</a><br />Patreon：<a href="https://www.patreon.com/Hijinzou" target="_blank">https://www.patreon.com/Hijinzou</a>`

	s2 := `Hi I'm Rynn! I can't speak much Japanese but I would still like to share my works <3

Tumblr: http://midorynn.tumblr.com/
Twitter: https://twitter.com/rynn_apple
Instagram: https://www.instagram.com/rynn_apple/`

	t1 := time.Now()
	n, _ := html.ParseFragment(strings.NewReader(s),
		&html.Node{
			Type:     html.ElementNode,
			Data:     "body",
			DataAtom: atom.Body,
		})
	b := &strings.Builder{}

	for _, n := range n {
		d := goquery.NewDocumentFromNode(n)
		d.Find("a").Each(func(i int, s *goquery.Selection) {
			if x, ok := s.Attr("href"); ok && strings.HasPrefix(x, "pixiv://") {
				s.RemoveAttr("href")
				s.SetAttr("to-pixiv", strings.TrimPrefix(x, "pixiv://"))
			}
		})
		h, _ := goquery.OuterHtml(d.Selection)
		b.Write([]byte(strings.TrimSpace(h)))
	}

	log.Print(b, b.String() == s, time.Now().Sub(t1))

	log.Print(strings.ReplaceAll(html.EscapeString(s2), "\n", "<br />"))
}

func testTags() {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	mdb := client.Database("bowerbird_test")
	ensureIndexes(ctx, mdb)
	mcp := mdb.Collection(m.CollectionPost)
	mct := mdb.Collection(m.CollectionTag)

	tids, err := mcp.Distinct(ctx, "tagIDs", bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(tids[:10])
	for _, x := range tids {
		if id, ok := x.(primitive.ObjectID); ok {
			n, err := mct.CountDocuments(ctx, bson.M{"_id": id})
			if err != nil {
				log.Fatal(err)
			}
			if n <= 0 {
				log.Fatal(x)
			}
		}
	}
}

func testWebDav() {
	ctx := context.Background()
	fs := webdav.NewMemFS()
	http.Handle("/", &webdav.Handler{
		FileSystem: fs,
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("[%s]: %s, ERROR: %s\n", r.Method, r.URL, err)
			} else {
				log.Printf("[%s]: %s \n", r.Method, r.URL)
			}
		},
	})
	fs.Mkdir(ctx, "test", 0644)
	fs.Mkdir(ctx, "test?", 0644)
	fs.Mkdir(ctx, "test:", 0644)
	fs.Mkdir(ctx, "Test", 0644)
	fs.Mkdir(ctx, "Test/wewe", 0644)
	f, err := fs.OpenFile(ctx, "test:/qwq.txt", os.O_WRONLY, 0777)
	log.Print(err)
	if err == nil {
		f.Write([]byte("nmsl"))
	}
	log.Print(fs.Mkdir(ctx, "Testw/wewe", 0644))

	http.ListenAndServe("127.0.0.1:10233", nil)
}

type bbfs struct {
	root      string
	files     []string
	ms        map[string]string
	rootFiles *root
}

type root struct {
	files []string
}

func (d *root) Close() error {
	return nil
}

func (d *root) Read([]byte) (int, error) {
	log.Print("Read")
	return 0, os.ErrPermission
}

func (d *root) Seek(int64, int) (int64, error) {
	return 0, os.ErrPermission
}

func (d *root) Readdir(c int) ([]os.FileInfo, error) {
	s := make([]os.FileInfo, len(d.files))
	for i, x := range d.files {
		ss, err := os.Stat(x)
		if err != nil {
			log.Print(err)
			continue
		}
		s[i] = ss
	}
	return s, nil
}

func (d *root) Stat() (os.FileInfo, error) {
	log.Print("Stat")
	return os.Stat(".")
}

func (d *root) Write([]byte) (int, error) {
	return 0, os.ErrPermission
}

func (*bbfs) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return os.ErrPermission
}
func (*bbfs) RemoveAll(ctx context.Context, name string) error {
	return os.ErrPermission
}
func (*bbfs) Rename(ctx context.Context, oldName, newName string) error {
	return os.ErrPermission
}

func (fs *bbfs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	log.Print(name)
	switch name {
	case "/", "":
		return os.Stat(".")
	}
	f, ok := fs.ms[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return os.Stat(f)
}
func (fs *bbfs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	log.Print(name)
	switch name {
	case "/", "":
		return fs.rootFiles, nil
	}
	return os.Open(fs.ms[name])
}

func webDavShow(db *mongo.Database) {
	ctx := context.Background()
	cposts := db.Collection("posts")
	ids := []primitive.ObjectID{}
	for _, x := range []string{
		"5e6a5d3a7fa5f8b68d167a62",
		"5e6a5d3a7fa5f8b68d167a67",
	} {
		id, _ := primitive.ObjectIDFromHex(x)
		ids = append(ids, id)
	}
	cur, err := cposts.Aggregate(ctx, bson.A{
		bson.M{"$match": bson.M{"tagIDs": bson.M{"$in": ids}}},
		bson.M{"$lookup": bson.M{
			"from":         "users",
			"localField":   "ownerID",
			"foreignField": "_id", "as": "owner"}},
		bson.M{"$addFields": bson.M{"ownerIDs": "$owner.sourceID"}},
		bson.M{"$project": bson.M{"sourceID": 1, "ownerIDs": 1}},
	})
	if err != nil {
		log.Fatal(err)
	}
	type result struct {
		SourceID string   `bson:"sourceID"`
		OwnerIDs []string `bson:"ownerIDs"`
	}
	rs := []result{}
	cur.All(ctx, &rs)
	// log.Fatal(rs)
	ms := map[string]string{}

	for _, r := range rs {
		rg, err := filepath.Glob(fmt.Sprintf(`D:\PixivDownload/%s/%s*`, r.OwnerIDs[0], r.SourceID))
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range rg {
			ms["/"+filepath.Base(f)] = f
		}
		if len(rg) > 1 {
			log.Print(rg)
		}
	}
	files := make([]string, len(ms))
	i := 0
	for _, x := range ms {
		files[i] = x
		i++
	}
	// log.Fatal(ms, files)

	fs := &bbfs{root: `D:\PixivDownload`, ms: ms, files: files, rootFiles: &root{files: files}}

	http.Handle("/", &webdav.Handler{
		FileSystem: fs,
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("[%s]: %s, ERROR: %s\n", r.Method, r.URL, err)
			} else {
				log.Printf("[%s]: %s \n", r.Method, r.URL)
			}
		},
	})

	http.ListenAndServe("127.0.0.1:10233", nil)
}

func main() {
	// cli.New().Run(os.Args)
	// loadDB()
	// chTags()
	// testTags()
	// testWebDav()
	// testCaption()
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	db := client.Database("bowerbird_test")

	// webDavShow(db)
	// loadDB()
	ensureIndexes(ctx, db)
}
