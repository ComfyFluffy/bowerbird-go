package cli

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"time"

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

var (
	optsFOAIDOnly = options.FindOneAndUpdate().
			SetUpsert(true).SetReturnDocument(options.After).
			SetProjection(D{{"_id", 1}})
	optsUUpsert = options.Update().SetUpsert(true)
)

func connectToDB(ctx context.Context, uri string) (*mongo.Client, error) {
	log.G.Info("connecting to database...")
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
	cm := db.Collection(model.CollectionMedia)

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

	cm.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: D{{"url", 1}},
		},
	)
}

func buildRegexQuery(s string) primitive.Regex {
	return primitive.Regex{Options: "i", Pattern: "^" + regexp.QuoteMeta(s) + "$"}
}

func lookupObjectID(r bson.Raw) primitive.ObjectID {
	return r.Lookup("_id").ObjectID()
}

func insertMediaWithURL(ctx context.Context, cm *mongo.Collection, url string, height, width int) (primitive.ObjectID, error) {
	var updater interface{}
	if height != 0 && width != 0 {
		updater = D{{"$set", D{{"width", width}, {"height", height}}}}
	} else {
		updater = A{}
	}
	r, err := cm.FindOneAndUpdate(ctx,
		D{{"url", url}}, updater,
		optsFOAIDOnly).DecodeBytes()
	if err != nil {
		return primitive.ObjectID{}, err
	}
	return lookupObjectID(r), nil
}

func updateAvatars(ctx context.Context, db *mongo.Database, uid, url string) error {
	if url == "" || uid == "" {
		return nil
	}
	cu := db.Collection(model.CollectionUser)
	cm := db.Collection(model.CollectionMedia)

	id, err := insertMediaWithURL(ctx, cm, url, 0, 0)
	if err != nil {
		return err
	}
	_, err = cu.UpdateOne(ctx,
		D{{"source", "pixiv"}, {"sourceID", uid},
			{"avatarIDs",
				D{{"$ne", id}}}},
		D{{"$push",
			D{{"avatarIDs", id}}}})
	return err
}

func saveUserProfileToDB(ru *pixiv.RespUserDetail, db *mongo.Database) error {
	ctx := context.Background()
	cu := db.Collection(model.CollectionUser)
	cud := db.Collection(model.CollectionUserDetail)
	cm := db.Collection(model.CollectionMedia)

	u, ud := model.User{
		Extension: &model.ExtUser{Pixiv: &model.PixivUser{
			IsFollowed:           ru.User.IsFollowed,
			TotalFollowing:       ru.Profile.TotalFollowUsers,
			TotalIllustSeries:    ru.Profile.TotalIllustSeries,
			TotalIllusts:         ru.Profile.TotalIllusts,
			TotalManga:           ru.Profile.TotalManga,
			TotalNovelSeries:     ru.Profile.TotalNovelSeries,
			TotalNovels:          ru.Profile.TotalNovels,
			TotalPublicBookmarks: ru.Profile.TotalIllustBookmarksPublic,
		}},
	}, model.UserDetail{
		Name: ru.User.Name,
		Extension: &model.ExtUserDetail{Pixiv: &model.PixivUserProfile{
			Account:        ru.User.Account,
			Birth:          string(ru.Profile.Birth),
			Region:         ru.Profile.CountryCode,
			Gender:         ru.Profile.Gender,
			IsPremium:      ru.Profile.IsPremium,
			TwitterAccount: ru.Profile.TwitterAccount,
			WebPage:        ru.Profile.Webpage,
			Bio:            ru.User.Comment,
			Workspace:      model.DD{},
		}},
	}

	if x := ru.Workspace["workspace_image_url"]; x != "" {
		id, err := insertMediaWithURL(ctx, cm, x, 0, 0)
		if err != nil {
			return err
		}
		ud.Extension.Pixiv.WorkspaceMediaID = id
	}
	delete(ru.Workspace, "workspace_image_url")
	for k, v := range ru.Workspace {
		ud.Extension.Pixiv.Workspace = append(ud.Extension.Pixiv.Workspace, bson.E{k, v})
	}
	sort.Sort(ud.Extension.Pixiv.Workspace)

	if ru.Profile.IsUsingCustomProfileImage && ru.Profile.BackgroundImageURL != "" {
		r, err := insertMediaWithURL(ctx, cm, ru.Profile.BackgroundImageURL, 0, 0)
		if err != nil {
			return err
		}
		ud.Extension.Pixiv.BackgroundMediaID = r
	}

	uid := strconv.Itoa(ru.User.ID)
	r, err := cu.FindOneAndUpdate(ctx,
		D{{"source", "pixiv"}, {"sourceID", uid}},
		D{
			{"$set", &u},
			{"$currentDate", D{{"lastModified", true}}},
		},
		optsFOAIDOnly,
	).DecodeBytes()
	if err != nil {
		return err
	}

	err = updateAvatars(ctx, db, uid, ru.User.ProfileImageURLs.Medium)
	if err != nil {
		return err
	}

	ud.UserID = lookupObjectID(r)
	_, err = cud.UpdateOne(ctx, &ud, A{}, optsUUpsert)
	if err != nil {
		return err
	}
	return nil
}

func saveIllustsToDB(ils []*pixiv.Illust, db *mongo.Database, usersToUpdate map[int]bool) error {
	ctx := context.Background()
	cu := db.Collection(model.CollectionUser)
	cp := db.Collection(model.CollectionPost)
	cpd := db.Collection(model.CollectionPostDetail)
	ct := db.Collection(model.CollectionTag)
	cm := db.Collection(model.CollectionMedia)

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
				Type:        il.Type,
				CaptionHTML: il.Caption,
				Title:       il.Title,
			}},
			MediaIDs: make([]primitive.ObjectID, 0, il.PageCount),
			Date:     il.CreateDate,
		}

		if il.MetaSinglePage.OriginalImageURL != "" {
			id, err := insertMediaWithURL(ctx, cm, il.MetaSinglePage.OriginalImageURL, il.Height, il.Width)
			if err != nil {
				return err
			}
			pd.MediaIDs = append(pd.MediaIDs, id)
		} else {
			for i, img := range il.MetaPages {
				var w, h int
				if i == 0 {
					h = il.Height
					w = il.Width
				}
				id, err := insertMediaWithURL(ctx, cm, img.ImageURLs.Original, h, w)
				if err != nil {
					return err
				}
				pd.MediaIDs = append(pd.MediaIDs, id)
			}
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
						D{{"source", "pixiv"}, {"alias", D{{"$in", treg}}}},
						D{{"$addToSet", D{
							{"alias", D{
								{"$each", ts}}}}}},
						optsFOAIDOnly).DecodeBytes()
				} else if len(ts) == 1 {
					r, err = ct.FindOneAndUpdate(ctx,
						D{{"source", "pixiv"}, {"alias", buildRegexQuery(ts[0])}},
						D{{"$setOnInsert", D{{"alias", ts}}}},
						optsFOAIDOnly).DecodeBytes()
				}
				if err != nil {
					return err
				}
				p.TagIDs = append(p.TagIDs, lookupObjectID(r))
			}
		}

		u := model.User{
			Extension: &model.ExtUser{Pixiv: &model.PixivUser{IsFollowed: il.User.IsFollowed}},
		}
		uid := strconv.Itoa(il.User.ID)
		r, err := cu.FindOneAndUpdate(ctx,
			D{{"source", "pixiv"}, {"sourceID", uid}},
			D{{"$set", u}},
			optsFOAIDOnly.SetProjection(D{{"_id", 1}, {"lastModified", 1}})).DecodeBytes()
		if err != nil {
			return err
		}
		p.OwnerID = lookupObjectID(r)
		if t, ok := r.Lookup("lastModified").TimeOK(); !ok || time.Now().Sub(t) > 120*time.Hour {
			usersToUpdate[il.User.ID] = true
		}
		err = updateAvatars(ctx, db, uid, il.User.ProfileImageURLs.Medium)
		if err != nil {
			return err
		}

		r, err = cp.FindOneAndUpdate(ctx,
			D{{"source", "pixiv"}, {"sourceID", p.SourceID}},
			D{{"$set", &p}, {"$currentDate", D{{"lastModified", true}}}},
			optsFOAIDOnly).DecodeBytes()
		_, err = cpd.UpdateOne(ctx, &pd, D{{"$set", D{{"postID", lookupObjectID(r)}}}}, optsUUpsert)
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
