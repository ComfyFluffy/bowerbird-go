package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User defines the person
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Rating    int                `bson:"rating,omitempty" json:"rating,omitempty"`
	Source    string             `bson:"source,omitempty" json:"source"`
	SourceID  string             `bson:"sourceID,omitempty" json:"sourceID"`
	Extension ExtUser            `bson:"extension,omitempty" json:"extension,omitempty"`

	AvatarID primitive.ObjectID   `bson:"avatar,omitempty" json:"-"`
	TagIDs   []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`

	Avatar *Media `bson:"-" json:"avatar,omitempty"`
	Tags   []Tag  `bson:"-" json:"tags"`
}

const CollectionUser = "users"

// DBCollection returns the name of mongodb collection
// func (User) DBCollection() string {
// 	return "users"
// }

type ExtUser struct {
	Pixiv *PixivUser `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// UserDetail defines user details
type UserDetail struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	UserID primitive.ObjectID `bson:"userID,omitempty" json:"-"`
	Name   string             `bson:"name,omitempty" json:"name,omitempty"`
	Page   *Page              `bson:"page,omitempty" json:"page"`
	// Banner   *Media                 `bson:"banner,omitempty" json:"banner,omitempty"`
	Extension *ExtUserDetail `bson:"extension,omitempty" json:"extension"`
}

// DBCollection returns the name of mongodb collection
const CollectionUserDetail = "user_profile"

type ExtUserDetail struct {
	Pixiv *PixivUserProfile `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// Post defines user created content
type Post struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Rating        int                `bson:"rating,omitempty" json:"rating,omitempty"`
	Source        string             `bson:"source,omitempty" json:"source,omitempty"`
	SourceID      string             `bson:"sourceID,omitempty" json:"sourceID,omitempty"`
	SourceDeleted bool               `bson:"sourceDeleted" json:"sourceDeleted"`
	SourcePrivate bool               `bson:"sourcePrivate" json:"sourcePrivate"`
	Extension     *ExtPost           `bson:"extension,omitempty" json:"extension"`

	OwnerID primitive.ObjectID   `bson:"ownerID,omitempty" json:"-"`
	TagIDs  []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`

	Owner User  `bson:"-" json:"owner,omitempty"`
	Tags  []Tag `bson:"-" json:"tags,omitempty"`
}

// DBCollection returns the name of mongodb collection
const CollectionPost = "posts"

type ExtPost struct {
	Pixiv *PixivPost `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// PostDetail defines the detail of Post
type PostDetail struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	PostID    primitive.ObjectID `bson:"postID,omitempty" json:"-"`
	Date      time.Time          `bson:"date,omitempty" json:"date,omitempty"`
	Page      *Page              `bson:"page,omitempty" json:"page"`
	Extension *ExtPostDetail     `bson:"extension,omitempty" json:"extension"`
}

type ExtPostDetail struct {
	Pixiv *PixivIllustDetail `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

const CollectionPostDetail = "post_details"

// Collection defines the collection of Post
type Collection struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Name      string             `bson:"name,omitempty" json:"name"`
	Extension *ExtCollection     `bson:"extension,omitempty" json:"extension"`

	TagIDs   []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`
	PostIDs  []primitive.ObjectID `bson:"postIDs,omitempty" json:"-"`
	MediaIDs []primitive.ObjectID `bson:"mediaIDs,omitempty" json:"-"`

	Tags  []Tag   `bson:"-" json:"tags"`
	Posts []Post  `bson:"-" json:"posts"`
	Media []Media `bson:"-" json:"media"`
}

const CollectionCollection = "collection"

type ExtCollection struct{}

// Tag defines the tag of the User, Post and Collection
type Tag struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Alias []TagAlias         `bson:"alias,omitempty" json:"alias,omitempty"`
}

const CollectionTag = "tags"

type TagAlias struct {
	Text     string `bson:"text,omitempty" json:"text,omitempty"`
	Language string `bson:"language,omitempty" json:"language,omitempty"`
	Source   string `bson:"source,omitempty" json:"source,omitempty"`
}

// Media defines the assets of Post
type Media struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	MIME      string             `bson:"mime,omitempty" json:"mime"`
	Colors    []Color            `bson:"colors,omitempty" json:"colors"`
	Size      int                `bson:"size,omitempty" json:"size"`
	URL       string             `bson:"url,omitempty" json:"-"`
	Path      string             `bson:"path,omitempty" json:"-"`
	Extension *ExtMedia          `bson:"extension,omitempty" json:"extension"`
}

const CollectionMedia = "media"

type ExtMedia struct {
	Pixiv *PixivMedia `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// Color defines the color of Media
type Color struct {
	R, G, B uint8
}

// Page defines the page of Post and User
type Page struct {
	HTML      string               `bson:"html,omitempty" json:"html"`
	MediaIDs  []primitive.ObjectID `bson:"mediaIDs,omitempty" json:"-"`
	Media     []Media              `bson:"-" json:"media"`
	Extension *ExtPage             `bson:"extension,omitempty" json:"extension"`
}

type ExtPage struct{}
