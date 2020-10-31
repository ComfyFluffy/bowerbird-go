package model

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureIndexes creates the MongoDB indexes.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	cu := db.Collection(CollectionUser)
	cud := db.Collection(CollectionUserDetail)
	ct := db.Collection(CollectionTag)
	cp := db.Collection(CollectionPost)
	cpd := db.Collection(CollectionPostDetail)
	cm := db.Collection(CollectionMedia)
	cc := db.Collection(CollectionCollection)

	_, err := cu.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: d{{Key: "source", Value: 1}, {Key: "sourceID", Value: 1}},
			Options: options.Index().
				SetUnique(true),
		},
	)
	if err != nil {
		return err
	}

	_, err = cud.Indexes().CreateMany(
		ctx, []mongo.IndexModel{
			{
				Keys: d{{Key: "userID", Value: 1}},
			},
			{
				Keys: d{{Key: "name", Value: 1}},
			},
		},
	)
	if err != nil {
		return err
	}

	_, err = cp.Indexes().CreateMany(
		ctx, []mongo.IndexModel{
			{
				Keys:    d{{Key: "source", Value: 1}, {Key: "sourceID", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys: d{{Key: "tagIDs", Value: 1}},
			},
		},
	)
	if err != nil {
		return err
	}

	_, err = cpd.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: d{{Key: "postID", Value: 1}},
		},
	)
	if err != nil {
		return err
	}

	_, err = ct.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: d{{Key: "alias", Value: 1}, {Key: "source", Value: 1}},
		},
	)
	if err != nil {
		return err
	}

	_, err = cm.Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys:    d{{Key: "url", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	)
	if err != nil {
		return err
	}

	_, err = cc.Indexes().CreateMany(
		ctx, []mongo.IndexModel{
			{
				Keys: d{{Key: "source", Value: 1}, {Key: "sourceID", Value: 1}},
			},
			{
				Keys: d{{Key: "tagIDs", Value: 1}},
			},
		},
	)
	return err
}
