package pixiv

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/WOo0W/bowerbird/cli/log"
	"github.com/WOo0W/bowerbird/helper/orderedmap"
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
	optsFUIDOnly = options.FindOneAndUpdate().
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
		optsFUIDOnly).DecodeBytes()
	if err != nil {
		return primitive.ObjectID{}, err
	}
	return lookupObjectID(r), nil
}

func updatePixivAvatars(ctx context.Context, cu, cm *mongo.Collection, uid string, url string) error {
	if url == "" || uid == "" {
		return nil
	}
	id, err := insertMediaWithURL(ctx, cm, model.MediaPixivAvatar, url, 0, 0)
	if err != nil {
		return err
	}
	_, err = cu.UpdateOne(ctx,
		d{{Key: "source", Value: model.SourcePixiv}, {Key: "sourceID", Value: uid}},
		d{
			{Key: "$addToSet", Value: d{{Key: "avatarIDs", Value: id}}},
			{Key: "$set", Value: d{{Key: "currentAvatarID", Value: id}}},
		})
	return err
}

func saveUserProfile(ctx context.Context, cu, cud, cm *mongo.Collection, ru *pixiv.RespUserDetail) error {
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
			Workspace:      orderedmap.O{},
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
		d{{Key: "source", Value: model.SourcePixiv}, {Key: "sourceID", Value: uid}},
		d{
			{Key: "$set", Value: &u},
			{Key: "$currentDate", Value: d{{Key: "lastModified", Value: true}}},
		},
		optsFUIDOnly,
	).DecodeBytes()
	if err != nil {
		return err
	}

	err = updatePixivAvatars(ctx, cu, cm, uid, ru.User.ProfileImageURLs.Medium)
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

func loadPixivTags(ctx context.Context, ct *mongo.Collection, tags []pixiv.Tag) ([]primitive.ObjectID, error) {
	oids := make([]primitive.ObjectID, 0, len(tags))
	for _, t := range tags {
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
					d{{Key: "source", Value: model.SourcePixiv}, {Key: "alias", Value: d{{Key: "$in", Value: ts}}}},
					d{{Key: "$addToSet", Value: d{
						{Key: "alias", Value: d{
							{Key: "$each", Value: ts}}}}}},
					optsFUIDOnly).DecodeBytes()
			} else if len(ts) == 1 {
				r, err = ct.FindOneAndUpdate(ctx,
					d{{Key: "source", Value: model.SourcePixiv}, {Key: "alias", Value: ts[0]}},
					d{{Key: "$setOnInsert", Value: d{{Key: "alias", Value: ts}}}},
					optsFUIDOnly).DecodeBytes()
			}
			if err != nil {
				return nil, err
			}
			oids = append(oids, lookupObjectID(r))
		}
	}
	return oids, nil
}

func saveIllusts(ils []*pixiv.Illust, cu, cp, cpd, ct, cm *mongo.Collection, usersToUpdate map[int]struct{}) error {
	ctx := context.Background()

	for _, il := range ils {
		sid := strconv.Itoa(il.ID)
		if !il.Visible {
			log.G.Warn("skipped invisible item:", sid)
			err := updateInvisiblePost(ctx, model.PostSourcePixivIllust, sid, cp)
			if err != nil {
				return err
			}
			continue
		}

		p, pd := &model.Post{
			Extension: &model.ExtPost{Pixiv: &model.PixivPost{
				IsBookmarked:   il.IsBookmarked,
				TotalBookmarks: il.TotalBookmarks,
				TotalViews:     il.TotalView,
			}},
			Source:   model.PostSourcePixivIllust,
			SourceID: sid,
		}, &model.PostDetail{
			Extension: &model.ExtPostDetail{PixivIllust: &model.PixivIllustDetail{
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

		savePixivPostAndDetail(ctx, ct, cu, cm, cp, cpd, usersToUpdate, &il.User, p, pd, il.Tags)
	}
	return nil
}

func saveNovels(nos []*pixiv.Novel, cu, cp, cpd, ct, cm, cc *mongo.Collection, api *pixiv.AppAPI, usersToUpdate map[int]struct{}, processed, limit int) (int, error) {
	ctx := context.Background()
	for _, no := range nos {
		if limit != 0 && processed >= limit {
			return processed, nil
		}
		processed++

		sid := strconv.Itoa(no.ID)
		if !no.Visible {
			log.G.Warn("skipped invisible item:", sid)
			err := updateInvisiblePost(ctx, model.PostSourcePixivNovel, sid, cp)
			if err != nil {
				return processed - 1, err
			}
			continue
		}

		log.G.Info(fmt.Sprintf("Saving novel text: %s (%s)", no.Title, sid))
		nod, err := api.Novel.Text(no.ID)
		if err != nil {
			log.G.Error(err)
			continue
		}

		p, pd := &model.Post{
			Extension: &model.ExtPost{Pixiv: &model.PixivPost{
				IsBookmarked:   no.IsBookmarked,
				TotalBookmarks: no.TotalBookmarks,
				TotalViews:     no.TotalView,
			}},
			Source:   model.PostSourcePixivNovel,
			SourceID: sid,
			TagIDs:   make([]primitive.ObjectID, 0, len(no.Tags)),
		}, &model.PostDetail{
			Extension: &model.ExtPostDetail{PixivNovel: &model.PixivNovelDetail{
				CaptionHTML: no.Caption,
				Text:        nod.NovelText,
				Title:       no.Title,
			}},
			Date: no.CreateDate,
		}

		if no.ImageURLs.Large != "" {
			id, err := insertMediaWithURL(ctx, cm, model.MediaPixivNovelCover, no.ImageURLs.Large, 0, 0)
			if err != nil {
				return processed - 1, err
			}
			pd.MediaIDs = []primitive.ObjectID{id}
		}

		err = savePixivPostAndDetail(ctx, ct, cu, cm, cp, cpd, usersToUpdate, &no.User, p, pd, no.Tags)
		if err != nil {
			return processed - 1, err
		}
	}
	return processed, nil
}

func updateInvisiblePost(ctx context.Context, source model.PostSource, sid string, cp *mongo.Collection) error {
	_, err := cp.UpdateOne(
		ctx,
		d{{Key: "source", Value: source}, {Key: "sourceID", Value: sid}},
		d{{Key: "$set", Value: d{{Key: "sourceInvisible", Value: true}}}},
		optsUUpsert)
	return err
}

func loadPixivUser(ctx context.Context, cu *mongo.Collection, usersToUpdate map[int]struct{}, uid int, isFollowed bool, before time.Duration) (primitive.ObjectID, error) {
	sid := strconv.Itoa(uid)

	r, err := cu.FindOneAndUpdate(ctx,
		d{{Key: "source", Value: model.SourcePixiv}, {Key: "sourceID", Value: sid}},
		d{{Key: "$set", Value: d{{Key: "extension.pixiv.isFollowed", Value: isFollowed}}}},
		optsFUIDOnly.SetProjection(d{{Key: "_id", Value: 1}, {Key: "lastModified", Value: 1}})).DecodeBytes()
	if err != nil {
		return primitive.NilObjectID, err
	}
	oid := lookupObjectID(r)
	if t, ok := r.Lookup("lastModified").TimeOK(); !ok || time.Since(t) > before {
		usersToUpdate[uid] = struct{}{}
	}
	return oid, nil
}

func updatePostDetail(ctx context.Context, source model.PostSource, cp, cpd *mongo.Collection, p *model.Post, pd *model.PostDetail) error {
	r, err := cp.FindOneAndUpdate(ctx,
		d{{Key: "source", Value: source}, {Key: "sourceID", Value: p.SourceID}},
		d{{Key: "$set", Value: p}, {Key: "$currentDate", Value: d{{Key: "lastModified", Value: true}}}},
		optsFUIDOnly).DecodeBytes()
	if err != nil {
		return err
	}
	_, err = cpd.UpdateOne(ctx, pd,
		d{{Key: "$set", Value: d{{Key: "postID", Value: lookupObjectID(r)}}}},
		optsUUpsert)
	return err
}

func savePixivPostAndDetail(ctx context.Context, ct, cu, cm, cp, cpd *mongo.Collection, usersToUpdate map[int]struct{}, pixivUser *pixiv.User, p *model.Post, pd *model.PostDetail, tags []pixiv.Tag) error {
	var err error
	if len(tags) > 0 {
		p.TagIDs, err = loadPixivTags(ctx, ct, tags)
		if err != nil {
			return err
		}
	}

	p.OwnerID, err = loadPixivUser(ctx, cu, usersToUpdate, pixivUser.ID, pixivUser.IsFollowed, 240*time.Hour)
	if err != nil {
		return err
	}

	err = updatePixivAvatars(ctx, cu, cm, strconv.Itoa(pixivUser.ID), pixivUser.ProfileImageURLs.Medium)
	if err != nil {
		return err
	}

	err = updatePostDetail(ctx, p.Source, cp, cpd, p, pd)
	if err != nil {
		return err
	}

	return nil
}
