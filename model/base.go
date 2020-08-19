package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type (
	a = bson.A
	d = bson.D
)

// User defines the person
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Rating    int                `bson:"rating,omitempty" json:"rating,omitempty"`
	Source    string             `bson:"source,omitempty" json:"source"`
	SourceID  string             `bson:"sourceID,omitempty" json:"sourceID"`
	Extension *ExtUser           `bson:"extension,omitempty" json:"extension,omitempty"`

	LastModified time.Time `bson:"lastModified,omitempty" json:"lastModified,omitempty"`

	AvatarIDs []primitive.ObjectID `bson:"avatarIDs,omitempty" json:"-"`
	TagIDs    []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`

	Tags       []Tag       `bson:"tags,omitempty" json:"tags,omitempty"`
	Avatar     *Media      `bson:"avatar,omitempty" json:"avatar,omitempty"`
	UserDetail *UserDetail `bson:"userDetail,omitempty" json:"userDetail,omitempty"`
}

const CollectionUser = "users"

// DBCollection returns the name of mongodb collection
// func (User) DBCollection() string {
// 	return "users"
// }

type ExtUser struct {
	Pixiv *PixivUser `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// UserDetail defines user details with change history
type UserDetail struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userID,omitempty" json:"-"`
	Name      string             `bson:"name,omitempty" json:"name,omitempty"`
	Extension *ExtUserDetail     `bson:"extension,omitempty" json:"extension"`
}

// CollectionUserDetail defines the collection name in MongoDB
const CollectionUserDetail = "user_details"

type ExtUserDetail struct {
	Pixiv *PixivUserProfile `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// Post defines user created content
type Post struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Rating          int                `bson:"rating,omitempty" json:"rating,omitempty"`
	Source          string             `bson:"source,omitempty" json:"source,omitempty"`
	SourceID        string             `bson:"sourceID,omitempty" json:"sourceID,omitempty"`
	SourceInvisible bool               `bson:"sourceInvisible" json:"sourceInvisible"`
	Extension       *ExtPost           `bson:"extension,omitempty" json:"extension"`
	Language        string             `bson:"language,omitempty" json:"language,omitempty"`

	LastModified time.Time `bson:"lastModified,omitempty" json:"lastModified,omitempty"`

	ParentID primitive.ObjectID   `bson:"parent,omitempty" json:"-"`
	OwnerID  primitive.ObjectID   `bson:"ownerID,omitempty" json:"-"`
	TagIDs   []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`

	Tags       []Tag       `bson:"tags,omitempty" json:"tags,omitempty"`
	PostDetail *PostDetail `bson:"postDetail,omitempty" json:"postDetail,omitempty"`
	Owner      *User       `bson:"owner,omitempty" json:"owner,omitempty"`
}

// DBCollection returns the name of mongodb collection
const CollectionPost = "posts"

type ExtPost struct {
	Pixiv *PixivPost `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// PostDetail defines the detail of Post
type PostDetail struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID    primitive.ObjectID `bson:"postID,omitempty" json:"-"`
	Date      time.Time          `bson:"date,omitempty" json:"date,omitempty"`
	Extension *ExtPostDetail     `bson:"extension,omitempty" json:"extension"`

	MediaIDs []primitive.ObjectID `bson:"mediaIDs,omitempty" json:"-"`
}

type ExtPostDetail struct {
	Pixiv *PixivIllustDetail `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

const CollectionPostDetail = "post_details"

// Collection defines the collection of Post
type Collection struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name,omitempty" json:"name"`
	Extension *ExtCollection     `bson:"extension,omitempty" json:"extension"`

	TagIDs  []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`
	PostIDs []primitive.ObjectID `bson:"postIDs,omitempty" json:"-"`

	Tags  []Tag  `bson:"-" json:"tags"`
	Posts []Post `bson:"-" json:"posts"`
}

const CollectionCollection = "collection"

type ExtCollection struct{}

// Tag defines the tag of the User, Post and Collection
type Tag struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Alias  []string           `bson:"alias,omitempty" json:"alias,omitempty"`
	Source string             `bson:"source,omitempty" json:"source,omitempty"`
}

const CollectionTag = "tags"

// type TagAlias struct {
// 	Text     string `bson:"text,omitempty" json:"text,omitempty"`
// 	Language string `bson:"language,omitempty" json:"language,omitempty"`
// 	Source   string `bson:"source,omitempty" json:"source,omitempty"`
// }

const CollectionMedia = "media"

type MediaType string

const (
	MediaPixivAvatar            MediaType = "pixiv-avatar"
	MediaPixivWorkspaceImage              = "pixiv-workspace-image"
	MediaPixivIllust                      = "pixiv-illust"
	MediaPixivProfileBackground           = "pixiv-profile-background"
)

// Media defines the assets of Post
type Media struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Type      MediaType          `bson:"type,omitempty" json:"type,omitempty"`
	MIME      string             `bson:"mime,omitempty" json:"mime"`
	Colors    []Color            `bson:"colors,omitempty" json:"colors"`
	Size      int                `bson:"size,omitempty" json:"size"`
	Height    int                `bson:"height,omitempty" json:"height,omitempty"`
	Width     int                `bson:"width,omitempty" json:"width,omitempty"`
	URL       string             `bson:"url,omitempty" json:"-"`
	Path      string             `bson:"path,omitempty" json:"-"`
	Extension *ExtMedia          `bson:"extension,omitempty" json:"extension"`
}

type ExtMedia struct {
	Pixiv *PixivMedia `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// Color defines the color of Media
type Color struct {
	R, G, B uint8
	Rating  int
}
