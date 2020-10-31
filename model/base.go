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

// Source stores the source of data
type Source string

// Various sources (websites)
const (
	SourcePixiv Source = "pixiv"
)

// User defines the creator of the content.
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Rating    int                `bson:"rating,omitempty" json:"rating,omitempty"`
	Source    Source             `bson:"source,omitempty" json:"source"`
	SourceID  string             `bson:"sourceID,omitempty" json:"sourceID"`
	Extension *ExtUser           `bson:"extension,omitempty" json:"extension,omitempty"`

	LastModified time.Time `bson:"lastModified,omitempty" json:"lastModified,omitempty"`

	CurrentAvatarID primitive.ObjectID   `bson:"currentAvatarID,omitempty" json:"-"`
	AvatarIDs       []primitive.ObjectID `bson:"avatarIDs,omitempty" json:"-"`
	TagIDs          []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`

	Tags       []Tag       `bson:"tags,omitempty" json:"tags,omitempty"`
	Avatar     *Media      `bson:"avatar,omitempty" json:"avatar,omitempty"`
	UserDetail *UserDetail `bson:"userDetail,omitempty" json:"userDetail,omitempty"`
}

// collection names in MongoDB
const (
	CollectionUser       = "users"
	CollectionUserDetail = "user_details"
	CollectionPost       = "posts"
	CollectionPostDetail = "post_details"
	CollectionCollection = "collection"
	CollectionTag        = "tags"
	CollectionMedia      = "media"
)

// ExtUser extends the User.
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

// ExtUserDetail extends the UserDetail.
type ExtUserDetail struct {
	Pixiv *PixivUserProfile `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// PostSource stores various sources of Post
type PostSource string

// Source of Post
const (
	PostSourcePixivIllust PostSource = "pixiv-illust"
	PostSourcePixivNovel  PostSource = "pixiv-novel"
)

// Post defines user created content.
type Post struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Rating          int                `bson:"rating,omitempty" json:"rating,omitempty"`
	Source          PostSource         `bson:"source,omitempty" json:"source,omitempty"`
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

// ExtPost extends the Post.
type ExtPost struct {
	Pixiv *PixivPost `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// PostDetail defines the detail of Post.
type PostDetail struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID    primitive.ObjectID `bson:"postID,omitempty" json:"-"`
	Date      time.Time          `bson:"date,omitempty" json:"date,omitempty"`
	Extension *ExtPostDetail     `bson:"extension,omitempty" json:"extension"`

	MediaIDs []primitive.ObjectID `bson:"mediaIDs,omitempty" json:"-"`
}

// ExtPostDetail extends the PostDetail from various sources
type ExtPostDetail struct {
	PixivIllust *PixivIllustDetail `bson:"pixivIllust,omitempty" json:"pixivIllust,omitempty"`
	PixivNovel  *PixivNovelDetail  `bson:"pixivNovel,omitempty" json:"pixivNovel,omitempty"`
}

// CollectionSource stores the sources of various collections.
type CollectionSource string

// Various CollectionSource
const (
	CollectionSourcePixivNovelSeries CollectionSource = "pixiv-novel-series"
)

// Collection defines the collection of Post.
type Collection struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Source    CollectionSource   `bson:"source,omitempty" json:"source,omitempty"`
	SourceID  string             `bson:"sourceID,omitempty" json:"sourceID,omitempty"`
	Name      string             `bson:"name,omitempty" json:"name"`
	Extension *ExtCollection     `bson:"extension,omitempty" json:"extension"`

	TagIDs  []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`
	PostIDs []primitive.ObjectID `bson:"postIDs,omitempty" json:"-"`

	Tags  []Tag  `bson:"-" json:"tags"`
	Posts []Post `bson:"-" json:"posts"`
}

// ExtCollection extends the Collection.
type ExtCollection struct{}

// Tag defines the tag of the User, Post and Collection.
type Tag struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Alias  []string           `bson:"alias,omitempty" json:"alias,omitempty"`
	Source Source             `bson:"source,omitempty" json:"source,omitempty"`
}

// MediaType stores the source and type of the media
type MediaType string

// media types
const (
	MediaPixivAvatar            MediaType = "pixiv-avatar"
	MediaPixivWorkspaceImage    MediaType = "pixiv-workspace-image"
	MediaPixivIllust            MediaType = "pixiv-illust"
	MediaPixivNovelCover        MediaType = "pixiv-novel-cover"
	MediaPixivProfileBackground MediaType = "pixiv-profile-background"
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

// ExtMedia extends the media from various sources
type ExtMedia struct {
	Pixiv *PixivMedia `bson:"pixiv,omitempty" json:"pixiv,omitempty"`
}

// Color defines the color of Media
type Color struct {
	R, G, B uint8
	Rating  int
}
