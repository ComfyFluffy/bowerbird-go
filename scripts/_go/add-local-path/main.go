package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/WOo0W/bowerbird/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func addPathField() {
	ctx := context.Background()
	var db *mongo.Database

	cm := db.Collection(model.CollectionMedia)
	cu := db.Collection(model.CollectionUser)
	cp := db.Collection(model.CollectionPost)
	cpd := db.Collection(model.CollectionPostDetail)

	r, err := cpd.Find(ctx, bson.D{{"mediaIDs.0", bson.D{{"$exists", true}}}})
	if err != nil {
		return err
	}
	pdAll := []model.PostDetail{}
	err = r.All(ctx, &pdAll)
	if err != nil {
		return err
	}

	basePath := conf.Storage.ParsedPixiv()

	var PximgDate = regexp.MustCompile(
		`(\d{4}/\d{2}/\d{2}/\d{2}/\d{2}/\d{2})`,
	)

	for _, pd := range pdAll {
		for _, mid := range pd.MediaIDs {
			m := model.Media{}
			err := cm.FindOne(ctx, bson.D{{"_id", mid}}).
				Decode(&m)
			if err != nil {
				return err
			}

			p := model.Post{}
			err = cp.FindOne(ctx, bson.D{{"_id", pd.PostID}}).
				Decode(&p)
			if err != nil {
				return err
			}

			b, err := cu.FindOne(ctx, bson.D{{"_id", p.OwnerID}}).DecodeBytes()
			if err != nil {
				return err
			}

			var fp, lf string

			if len(pd.MediaIDs) == 1 {
				// continue
				fn := filepath.Base(m.URL)
				i := strings.LastIndexByte(fn, '.')
				fp = b.Lookup("sourceID").StringValue() + "/" +
					fn[:i] + "_" + strings.ReplaceAll(PximgDate.FindString(m.URL), "/", "") + fn[i:]
				lf = filepath.Join(basePath, fp)
			} else {
				fp = b.Lookup("sourceID").StringValue() + "/" +
					p.SourceID + "_" +
					strings.ReplaceAll(PximgDate.FindString(m.URL), "/", "") + "/" +
					filepath.Base(m.URL)
				lf = filepath.Join(basePath, fp)
			}

			if _, err := os.Stat(lf); os.IsNotExist(err) {
				continue
			}
			// log.G.Info(fmt.Sprintf("m.ID %s fp %s lf %s m %+v p %+v pd %+v", m.ID, fp, lf, m, p, pd))

			// return nil
			_, err = cm.UpdateOne(ctx,
				bson.D{{"_id", m.ID}},
				bson.D{{"$set", bson.D{{"path", fp}}}},
			)
			if err != nil {
				return err
			}

		}

	}

	return nil
}

func pathCheck() {
	ctx := context.Background()
	var db *mongo.Database
	r, err := db.Collection(model.CollectionMedia).Find(ctx, bson.D{{"path", bson.D{{"$exists", true}}}})
	if err != nil {
		return err
	}
	x := conf.Storage.ParsedPixiv()

	a := []model.Media{}
	r.All(ctx, &a)
	for _, aa := range a {
		if _, err := os.Stat(filepath.Join(x, aa.Path)); os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
