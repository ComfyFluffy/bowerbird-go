package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Source   string             `bson:",omitempty" json:"source"`
	SourceID string             `bson:",omitempty" json:"sourceID"`

	Tags   []Tag                `bson:"-" json:"tags"`
	TagIDs []primitive.ObjectID `bson:",omitempty" json:"-"`
}

// DBCollection returns the name of mongodb collection.
func (*User) DBCollection() string {
	return "users"
}

type Item struct {
	ID       primitive.ObjectID   `bson:"_id,omitempty" json:"id,string"`
	Source   string               `bson:",omitempty" json:"source,omitempty"`
	SourceID string               `bson:",omitempty" json:"sourceID,omitempty"`
	Rating   int                  `bson:",omitempty" json:"rating,omitempty"`
	OwnerID  primitive.ObjectID   `bson:",omitempty" json:"-"`
	TagsIDs  []primitive.ObjectID `bson:",omitempty" json:"-"`

	Owner User  `bson:"-" json:"owner,omitempty"`
	Tags  []Tag `bson:"-" json:"tags,omitempty"`
}

// DBCollection returns the name of mongodb collection.
func (*Item) DBCollection() string {
	return "items"
}

type ItemDetail struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	ItemID primitive.ObjectID `bson:",omitempty" json:"-"`
	Page   string             `bson:",omitempty" json:"page,omitempty"`
	Date   time.Time          `bson:",omitempty" json:"date"`
	Images []Image            `bson:",omitempty" json:"images,omitempty"`
}

// DBCollection returns the name of mongodb collection.
func (*ItemDetail) DBCollection() string {
	return "itemDetails"
}

// type Collection struct {
// 	ID     primitive.ObjectID   `bson:"_id,omitempty" json:"id,string"`
// 	Name   string               `json:"name"`
// 	TagIDs []primitive.ObjectID `bson:",omitempty" json:"-"`

// 	Items []Item `bson:"-" json:"items"`
// }

// // DBCollection returns the name of mongodb collection.
// func (*Collection) DBCollection() string {
// 	return "users"
// }

type Tag struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Alias []string           `bson:",omitempty" json:"alias,omitempty"`
}

// DBCollection returns the name of mongodb collection.
func (*Tag) DBCollection() string {
	return "tags"
}

// type Comment struct {
// 	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
// 	AuthorID primitive.ObjectID `json:"-"`
// 	Text     string             `json:"text"`

// 	Author User `bson:"-" json:"author,omitempty"`
// }

type Image struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Format string             `bson:",omitempty" json:"format"`
}
