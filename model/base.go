package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User defines the person
type User struct {
	ID       primitive.ObjectID     `bson:"_id,omitempty" json:"id,string"`
	Rating   int                    `bson:"rating,omitempty" json:"rating,omitempty"`
	Source   string                 `bson:"source,omitempty" json:"source"`
	SourceID string                 `bson:"sourceID,omitempty" json:"sourceID"`
	Extended map[string]interface{} `bson:"extended,omitempty" json:"extended"`

	TagIDs []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`

	Tags []Tag `bson:"-" json:"tags"`
}

// DBCollection returns the name of mongodb collection
func (User) DBCollection() string {
	return "users"
}

// UserProfile defines user details
type UserProfile struct {
	ID       primitive.ObjectID     `bson:"_id,omitempty" json:"id,string"`
	UserID   primitive.ObjectID     `bson:"userID,omitempty" json:"-"`
	Page     *Page                  `bson:"page,omitempty" json:"page"`
	Banner   *Media                 `bson:"banner,omitempty" json:"banner,omitempty"`
	Extended map[string]interface{} `bson:"extended,omitempty" json:"extended"`
}

// DBCollection returns the name of mongodb collection
func (UserProfile) DBCollection() string {
	return "user_profile"
}

// Post defines user created content
type Post struct {
	ID              primitive.ObjectID     `bson:"_id,omitempty" json:"id,string"`
	Rating          int                    `bson:"rating,omitempty" json:"rating,omitempty"`
	Source          string                 `bson:"source,omitempty" json:"source,omitempty"`
	SourceID        string                 `bson:"sourceID,omitempty" json:"sourceID,omitempty"`
	SourceInvisible bool                   `bson:"sourceInvisible" json:"sourceInvisible"`
	Extended        map[string]interface{} `bson:"extended,omitempty" json:"extended"`

	OwnerID primitive.ObjectID   `bson:"ownerID,omitempty" json:"-"`
	TagIDs  []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`

	Owner User  `bson:"-" json:"owner,omitempty"`
	Tags  []Tag `bson:"-" json:"tags,omitempty"`
}

// DBCollection returns the name of mongodb collection
func (Post) DBCollection() string {
	return "posts"
}

// PostDetail defines the detail of Post
type PostDetail struct {
	ID       primitive.ObjectID     `bson:"_id,omitempty" json:"id,string"`
	PostID   primitive.ObjectID     `bson:"postID,omitempty" json:"-"`
	Date     time.Time              `bson:"date,omitempty" json:"date,omitempty"`
	Page     *Page                  `bson:"page,omitempty" json:"page"`
	Extended map[string]interface{} `bson:"extended,omitempty" json:"extended"`
}

// DBCollection returns the name of mongodb collection
func (PostDetail) DBCollection() string {
	return "post_details"
}

// Collection defines the collection of Post
type Collection struct {
	ID       primitive.ObjectID     `bson:"_id,omitempty" json:"id,string"`
	Name     string                 `bson:"name,omitempty" json:"name"`
	Extended map[string]interface{} `bson:"extended,omitempty" json:"extended"`

	TagIDs   []primitive.ObjectID `bson:"tagIDs,omitempty" json:"-"`
	PostIDs  []primitive.ObjectID `bson:"postIDs,omitempty" json:"-"`
	MediaIDs []primitive.ObjectID `bson:"mediaIDs,omitempty" json:"-"`

	Tags  []Tag   `bson:"-" json:"tags"`
	Posts []Post  `bson:"-" json:"posts"`
	Media []Media `bson:"-" json:"media"`
}

// DBCollection returns the name of mongodb collection
func (Collection) DBCollection() string {
	return "collection"
}

// Tag defines the tag of the User, Post and Collection
type Tag struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id,string"`
	Alias []string           `bson:"alias,omitempty" json:"alias,omitempty"`
}

// DBCollection returns the name of mongodb collection
func (Tag) DBCollection() string {
	return "tags"
}

// Media defines the assets of Post
type Media struct {
	ID       primitive.ObjectID     `bson:"_id,omitempty" json:"id,string"`
	MIME     string                 `bson:"mime,omitempty" json:"mime"`
	Colors   []Color                `bson:"colors,omitempty" json:"colors"`
	Size     int                    `bson:"size,omitempty" json:"size"`
	URL      string                 `bson:"url,omitempty" json:"-"`
	Path     string                 `bson:"path,omitempty" json:"-"`
	Extended map[string]interface{} `bson:"extended,omitempty" json:"extended"`
}

// DBCollection returns the name of mongodb collection
func (Media) DBCollection() string {
	return "media"
}

// Color defines the color of Media
type Color struct {
	R, G, B uint8
}

// Page defines the page of Post and User
type Page struct {
	HTML     string                 `bson:"html,omitempty" json:"html"`
	MediaIDs []primitive.ObjectID   `bson:"mediaIDs,omitempty" json:"-"`
	Media    []Media                `bson:"-" json:"media"`
	Extended map[string]interface{} `bson:"extended,omitempty" json:"extended"`
}
