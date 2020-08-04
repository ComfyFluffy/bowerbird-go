package model

import (
	"encoding/json"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DD bson.D

func (a DD) Len() int           { return len(a) }
func (a DD) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DD) Less(i, j int) bool { return a[i].Key < a[j].Key }
func (a DD) MarshalJSON() ([]byte, error) {
	return json.Marshal(bson.D(a).Map())
}

//TODO: better ways to unmarshal it?

func (a *DD) UnmarshalJSON(b []byte) error {
	var m bson.M
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	for k, v := range m {
		*a = append(*a, bson.E{Key: k, Value: v})
	}
	sort.Sort(a)
	return nil
}

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

	Tags []Tag `bson:"-" json:"tags"`
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
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userID,omitempty" json:"-"`
	Name      string             `bson:"name,omitempty" json:"name,omitempty"`
	Extension *ExtUserDetail     `bson:"extension,omitempty" json:"extension"`
}

// DBCollection returns the name of mongodb collection
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

	LastModified time.Time `bson:"lastModified,omitempty" json:"lastModified,omitempty"`

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

// Media defines the assets of Post
type Media struct {
	MIME      string    `bson:"mime,omitempty" json:"mime"`
	Colors    []Color   `bson:"colors,omitempty" json:"colors"`
	Size      int       `bson:"size,omitempty" json:"size"`
	Height    int       `bson:"height,omitempty" json:"height,omitempty"`
	Width     int       `bson:"width,omitempty" json:"width,omitempty"`
	URL       string    `bson:"url,omitempty" json:"-"`
	Path      string    `bson:"path,omitempty" json:"-"`
	Extension *ExtMedia `bson:"extension,omitempty" json:"extension"`
}

type ExtMedia struct {
	Pixiv *PixivMedia `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// Color defines the color of Media
type Color struct {
	R, G, B uint8
	Rating  int
}
