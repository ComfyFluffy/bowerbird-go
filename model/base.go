package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Rating   int                `bson:",omitempty" json:"rating,omitempty"`
	Source   string             `bson:",omitempty" json:"source"`
	SourceID string             `bson:",omitempty" json:"sourceID"`

	TagIDs []primitive.ObjectID `bson:",omitempty" json:"-"`

	Tags []Tag `bson:"-" json:"tags"`
}

// DBCollection returns the name of mongodb collection.
func (User) DBCollection() string {
	return "users"
}

type UserProfile struct {
	ID       primitive.ObjectID     `bson:"_id,omitempty" json:"id,string"`
	UserID   primitive.ObjectID     `bson:",omitempty" json:"-"`
	Page     Page                   `bson:",omitempty" json:"page"`
	Extended map[string]interface{} `bson:",omitempty" json:"extended"`
}

// DBCollection returns the name of mongodb collection.
func (UserProfile) DBCollection() string {
	return "user_profile"
}

type Post struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Rating          int                `bson:",omitempty" json:"rating,omitempty"`
	Source          string             `bson:",omitempty" json:"source,omitempty"`
	SourceID        string             `bson:",omitempty" json:"sourceID,omitempty"`
	SourceInvisible bool               `json:"sourceInvisible"`

	OwnerID primitive.ObjectID   `bson:",omitempty" json:"-"`
	TagsIDs []primitive.ObjectID `bson:",omitempty" json:"-"`

	Owner User  `bson:"-" json:"owner,omitempty"`
	Tags  []Tag `bson:"-" json:"tags,omitempty"`
}

// DBCollection returns the name of mongodb collection.
func (Post) DBCollection() string {
	return "posts"
}

type PostDetail struct {
	ID       primitive.ObjectID     `bson:"_id,omitempty" json:"id,string"`
	PostID   primitive.ObjectID     `bson:",omitempty" json:"-"`
	Date     time.Time              `bson:",omitempty" json:"date,omitempty"`
	Page     Page                   `bson:",omitempty" json:"page"`
	Extended map[string]interface{} `bson:",omitempty" json:"extended"`
}

// DBCollection returns the name of mongodb collection.
func (PostDetail) DBCollection() string {
	return "post_details"
}

type Collection struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Name string             `json:"name"`

	TagIDs  []primitive.ObjectID `bson:",omitempty" json:"-"`
	PostIDs []primitive.ObjectID `bson:",omitempty" json:"-"`

	Tags  []Tag  `bson:"-" json:"tags"`
	Posts []Post `bson:"-" json:"posts"`
}

// DBCollection returns the name of mongodb collection.
func (Collection) DBCollection() string {
	return "collection"
}

type Tag struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Alias []string           `bson:",omitempty" json:"alias,omitempty"`
}

// DBCollection returns the name of mongodb collection.
func (Tag) DBCollection() string {
	return "tags"
}

// type Comment struct {
// 	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
// 	AuthorID primitive.ObjectID `json:"-"`
// 	Text     string             `json:"text"`

// 	Author User `bson:"-" json:"author,omitempty"`
// }

type Media struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	MIME   string             `bson:",omitempty" json:"mime"`
	Colors []Color            `bson:",omitempty" json:"colors"`
	Size   int                `bson:",omitempty" json:"size"`
	URL    string             `bson:",omitempty" json:"-"`
	Path   string             `bson:",omitempty" json:"-"`
}

func (Media) DBCollection() string {
	return "media"
}

type Color struct {
	R, G, B uint8
}

type Page struct {
	Markdown string               `bson:",omitempty" json:"markdown"`
	MediaIDs []primitive.ObjectID `bson:",omitempty" json:"-"`
	Media    []Media              `bson:"-" json:"media"`
}
