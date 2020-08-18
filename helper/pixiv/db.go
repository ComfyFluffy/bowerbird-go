package pixiv

import (
	"context"
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
)

type (
	a = bson.A
	d = bson.D
)

var (
	optsFOAIDOnly = options.FindOneAndUpdate().
			SetUpsert(true).SetReturnDocument(options.After).
			SetProjection(d{{Key: "_id", Value: 1}})
	optsUUpsert = options.Update().SetUpsert(true)
)

func lookupObjectID(r bson.Raw) primitive.ObjectID {
	return r.Lookup("_id").ObjectID()
}

func insertMediaWithURL(ctx context.Context, cm *mongo.Collection, t model.MediaType, url string, height, width int) (primitive.ObjectID, error) {
	var updater interface{}
	if height != 0 && width != 0 {
		updater = d{{Key: "$set", Value: d{
			{Key: "width", Value: width},
			{Key: "height", Value: height},
			{Key: "type", Value: t},
		}}}
	} else {
		updater = d{{Key: "$set", Value: d{
			{Key: "type", Value: t},
		}}}
	}
	r, err := cm.FindOneAndUpdate(ctx,
		d{{Key: "url", Value: url}}, updater,
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

	id, err := insertMediaWithURL(ctx, cm, model.MediaPixivAvatar, url, 0, 0)
	if err != nil {
		return err
	}
	_, err = cu.UpdateOne(ctx,
		d{{Key: "source", Value: "pixiv"}, {Key: "sourceID", Value: uid},
			{Key: "avatarIDs",
				Value: d{{Key: "$ne", Value: id}}}},
		d{{Key: "$push",
			Value: d{{Key: "avatarIDs", Value: id}}}})
	return err
}

func saveUserProfile(ru *pixiv.RespUserDetail, db *mongo.Database) error {
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
		id, err := insertMediaWithURL(ctx, cm, model.MediaPixivWorkspaceImage, x, 0, 0)
		if err != nil {
			return err
		}
		ud.Extension.Pixiv.WorkspaceMediaID = id
	}
	delete(ru.Workspace, "workspace_image_url")
	for k, v := range ru.Workspace {
		ud.Extension.Pixiv.Workspace = append(ud.Extension.Pixiv.Workspace, bson.E{Key: k, Value: v})
	}
	sort.Sort(ud.Extension.Pixiv.Workspace)

	if ru.Profile.IsUsingCustomProfileImage && ru.Profile.BackgroundImageURL != "" {
		r, err := insertMediaWithURL(ctx, cm, model.MediaPixivProfileBackground, ru.Profile.BackgroundImageURL, 0, 0)
		if err != nil {
			return err
		}
		ud.Extension.Pixiv.BackgroundMediaID = r
	}

	uid := strconv.Itoa(ru.User.ID)
	r, err := cu.FindOneAndUpdate(ctx,
		d{{Key: "source", Value: "pixiv"}, {Key: "sourceID", Value: uid}},
		d{
			{Key: "$set", Value: &u},
			{Key: "$currentDate", Value: d{{Key: "lastModified", Value: true}}},
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
	_, err = cud.UpdateOne(ctx, &ud, a{}, optsUUpsert)
	if err != nil {
		return err
	}
	return nil
}

func saveIllusts(ils []*pixiv.Illust, db *mongo.Database, usersToUpdate map[int]struct{}) error {
	ctx := context.Background()
	cu := db.Collection(model.CollectionUser)
	cp := db.Collection(model.CollectionPost)
	cpd := db.Collection(model.CollectionPostDetail)
	ct := db.Collection(model.CollectionTag)
	cm := db.Collection(model.CollectionMedia)

	for _, il := range ils {
		sid := strconv.Itoa(il.ID)
		if !il.Visible {
			log.G.Warn("skipped invisible item:", il.ID)
			_, err := cp.UpdateOne(
				ctx,
				d{{Key: "source", Value: "pixiv"}, {Key: "sourceID", Value: sid}},
				d{{Key: "$set", Value: d{{Key: "sourceInvisible", Value: true}}}},
				optsUUpsert)
			if err != nil {
				return err
			}
			continue
		}

		p, pd := model.Post{
			Extension: &model.ExtPost{Pixiv: &model.PixivPost{
				IsBookmarked:   il.IsBookmarked,
				TotalBookmarks: il.TotalBookmarks,
				TotalViews:     il.TotalView,
			}},
			Source:   "pixiv",
			SourceID: sid,
			TagIDs:   make([]primitive.ObjectID, 0, len(il.Tags)),
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
			id, err := insertMediaWithURL(ctx, cm, model.MediaPixivIllust, il.MetaSinglePage.OriginalImageURL, il.Height, il.Width)
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
				id, err := insertMediaWithURL(ctx, cm, model.MediaPixivIllust, img.ImageURLs.Original, h, w)
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
					r, err = ct.FindOneAndUpdate(ctx,
						d{{Key: "source", Value: "pixiv"}, {Key: "alias", Value: d{{Key: "$in", Value: ts}}}},
						d{{Key: "$addToSet", Value: d{
							{Key: "alias", Value: d{
								{Key: "$each", Value: ts}}}}}},
						optsFOAIDOnly).DecodeBytes()
				} else if len(ts) == 1 {
					r, err = ct.FindOneAndUpdate(ctx,
						d{{Key: "source", Value: "pixiv"}, {Key: "alias", Value: ts[0]}},
						d{{Key: "$setOnInsert", Value: d{{Key: "alias", Value: ts}}}},
						optsFOAIDOnly).DecodeBytes()
				}
				if err != nil {
					return err
				}
				p.TagIDs = append(p.TagIDs, lookupObjectID(r))
			}
		}

		uid := strconv.Itoa(il.User.ID)
		r, err := cu.FindOneAndUpdate(ctx,
			d{{Key: "source", Value: "pixiv"}, {Key: "sourceID", Value: uid}},
			d{{Key: "$set", Value: d{{Key: "extension.pixiv.isFollowed", Value: il.User.IsFollowed}}}},
			optsFOAIDOnly.SetProjection(d{{Key: "_id", Value: 1}, {Key: "lastModified", Value: 1}})).DecodeBytes()
		if err != nil {
			return err
		}
		p.OwnerID = lookupObjectID(r)
		if t, ok := r.Lookup("lastModified").TimeOK(); !ok || time.Since(t) > 240*time.Hour {
			usersToUpdate[il.User.ID] = struct{}{}
		}
		err = updateAvatars(ctx, db, uid, il.User.ProfileImageURLs.Medium)
		if err != nil {
			return err
		}

		r, err = cp.FindOneAndUpdate(ctx,
			d{{Key: "source", Value: "pixiv"}, {Key: "sourceID", Value: p.SourceID}},
			d{{Key: "$set", Value: &p}, {Key: "$currentDate", Value: d{{Key: "lastModified", Value: true}}}},
			optsFOAIDOnly).DecodeBytes()
		_, err = cpd.UpdateOne(ctx, &pd,
			d{{Key: "$set", Value: d{{Key: "postID", Value: lookupObjectID(r)}}}},
			optsUUpsert)
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

func UpdateAllUsers(db *mongo.Database, api *pixiv.AppAPI, forceAll bool) error {
	ctx := context.Background()
	cu := db.Collection("users")
	var filter d
	if forceAll {
		filter = d{{Key: "source", Value: "pixiv"}}
	} else {
		filter = d{
			{Key: "source", Value: "pixiv"},
			{Key: "$or", Value: a{
				d{{Key: "lastModified", Value: d{{Key: "$exists", Value: false}}}},
				d{{Key: "lastModified", Value: d{{Key: "$lt", Value: time.Now().Add(-240 * time.Hour)}}}},
			}},
		}
	}
	cur, err := cu.Find(ctx,
		filter,
		options.Find().SetProjection(d{{Key: "sourceID", Value: 1}}))
	if err != nil {
		return err
	}
	ids := make([]int, 0, 1024)
	for cur.Next(ctx) {
		id := cur.Current.Lookup("sourceID").StringValue()
		idInt, err := strconv.Atoi(id)
		if err != nil {
			return err
		}
		ids = append(ids, idInt)
	}
	updateUsers(db, api, ids)
	return nil
}
