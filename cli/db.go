package cli

import (
	"context"
	"regexp"
	"strconv"

	"github.com/WOo0W/bowerbird/cli/log"
	"github.com/WOo0W/bowerbird/model"
	"github.com/WOo0W/go-pixiv/pixiv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type A = bson.A
type D = bson.D

func connectToDB(ctx context.Context, uri string) (*mongo.Client, error) {
	log.G.Info("connecting to database")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return client, err
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return client, err
	}
	return client, nil
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

func buildRegexQuery(s string) primitive.Regex {
	return primitive.Regex{Options: "i", Pattern: "^" + regexp.QuoteMeta(s) + "$"}
}

func saveIllustToDB(ils []*pixiv.Illust, db *mongo.Database) error {
	ctx := context.Background()
	cu := db.Collection(model.CollectionUser)
	cp := db.Collection(model.CollectionPost)
	cpd := db.Collection(model.CollectionPostDetail)
	ct := db.Collection(model.CollectionTag)

	opts := options.FindOneAndUpdate().
		SetUpsert(true).SetReturnDocument(options.After).
		SetProjection(D{{"_id", 1}})

	for _, il := range ils {

		p, pd := model.Post{
			Extension: &model.ExtPost{Pixiv: &model.PixivPost{
				IsBookmarked:   il.IsBookmarked,
				TotalBookmarks: il.TotalBookmarks,
				TotalViews:     il.TotalView,
			}},
			Source:          "pixiv",
			SourceID:        strconv.Itoa(il.ID),
			SourceInvisible: !il.Visible,
			TagIDs:          make([]primitive.ObjectID, 0, len(il.Tags)),
		}, model.PostDetail{
			Extension: &model.ExtPostDetail{Pixiv: &model.PixivIllustDetail{
				Type: il.Type,
			}},
			Date: il.CreateDate,
			Page: &model.Page{},
		}

		// update & find TagIDs
		for _, t := range il.Tags {
			ts := make([]string, 0, 2)
			if t.Name != "" {
				ts = append(ts, t.Name)
			}
			if t.TranslatedName != "" {
				ts = append(ts, t.TranslatedName)
			}

			if len(ts) > 0 {
				var (
					r   bson.Raw
					err error
				)
				if len(ts) > 1 {
					treg := make([]primitive.Regex, len(ts))
					for i, tt := range ts {
						// do the match with case ignored
						treg[i] = buildRegexQuery(tt)
					}
					r, err = ct.FindOneAndUpdate(ctx,
						D{{"source", "pixiv"}, {"alias", D{{"$in", ts}}}},
						D{{"$addToSet", D{
							{"alias", D{
								{"$each", ts}}}}}},
						opts).DecodeBytes()
				} else if len(ts) == 1 {
					r, err = ct.FindOneAndUpdate(ctx,
						D{{"source", "pixiv"}, {"alias", buildRegexQuery(ts[0])}},
						D{{"$setOnInsert", D{{"alias", ts}}}},
						opts).DecodeBytes()
				}
				if err != nil {
					return err
				}
				p.TagIDs = append(p.TagIDs, r.Lookup("_id").ObjectID())
			}
		}

		u := model.User{
			Extension: &model.ExtUser{Pixiv: &model.PixivUser{IsFollowed: il.User.IsFollowed}},
		}
		r, err := cu.FindOneAndUpdate(ctx,
			D{{"source", "pixiv"}, {"sourceID", strconv.Itoa(il.User.ID)}},
			D{{"$set", u}},
			opts).DecodeBytes()
		if err != nil {
			return err
		}
		p.OwnerID = r.Lookup("_id").ObjectID()

		r, err = cp.FindOneAndUpdate(ctx,
			D{{"source", "pixiv"}, {"sourceID", p.SourceID}},
			D{{"$set", &p}, {"$currentDate", D{{"lastModified", true}}}},
			opts).DecodeBytes()
		pd.PostID = r.Lookup("_id").ObjectID()
		_, err = cpd.UpdateOne(ctx, &pd, A{}, options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	}

	// r, err := cp.Find(ctx, D{{"source", "pixiv"}, {"sourceID", D{{"$in", ids}}}}, options.Find().SetProjection(D{{"_id", 1, ""}}))
	// for r.Next(ctx) {
	// 	r.Current.Lookup("_id").ObjectID()
	// }
	// if err != nil {
	// 	return err
	// }
	return nil
}
