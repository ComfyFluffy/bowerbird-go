package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// PixivUser extends User with Pixiv's user struct
type PixivUser struct {
	IsFollowed           bool `bson:"isFollowed" json:"isFollowed"`
	TotalFollowing       int  `bson:"totalFollowing,omitempty" json:"totalFollowing"`
	TotalPublicBookmarks int  `bson:"totalPublicBookmarks,omitempty" json:"totalPublicBookmarks"`
	TotalIllusts         int  `bson:"totalIllusts,omitempty" json:"totalIllusts"`
	TotalManga           int  `bson:"totalManga,omitempty" json:"totalManga"`
	TotalNovels          int  `bson:"totalNovels,omitempty" json:"totalNovels"`
	TotalIllustSeries    int  `bson:"totalIllustSeries,omitempty" json:"totalIllustSeries"`
	TotalNovelSeries     int  `bson:"totalNovelSeries,omitempty" json:"totalNovelSeries"`
}

// PixivUserProfile extends UserDetail with Pixiv's user struct
type PixivUserProfile struct {
	Account           string             `bson:"account,omitempty" json:"account"`
	IsPremium         bool               `bson:"isPremium" json:"isPremium"`
	Birth             string             `bson:"birth,omitempty" json:"birth,omitempty"`
	Region            string             `bson:"region,omitempty" json:"region,omitempty"`
	Gender            string             `bson:"gender,omitempty" json:"gender,omitempty"`
	TwitterAccount    string             `bson:"twitterAccount,omitempty" json:"twitterAccount,omitempty"`
	WebPage           string             `bson:"webPage,omitempty" json:"webPage,omitempty"`
	Bio               string             `bson:"bio,omitempty" json:"bio,omitempty"`
	Workspace         DD                 `bson:"workspace,omitempty" json:"workspace,omitempty"`
	WorkspaceMediaID  primitive.ObjectID `bson:"workspaceMediaID,omitempty" json:"-"`
	BackgroundMediaID primitive.ObjectID `bson:"backgroudMediaID,omitempty" json:"-"`
}

// PixivPost extends PostDetail with Pixiv's illust struct
type PixivPost struct {
	IsBookmarked   bool `bson:"isBookmarked" json:"isBookmarked"`
	TotalBookmarks int  `bson:"totalBookmarks,omitempty" json:"totalBookmarks,omitempty"`
	TotalViews     int  `bson:"totalView,omitempty" json:"totalView,omitempty"`
}

// PixivIllustDetail extends PostDetail with Pixiv's illust struct
type PixivIllustDetail struct {
	// Type can be "illust", "manga" or "novel"
	Type        string `bson:"type,omitempty" json:"type"`
	CaptionHTML string `bson:"captionHTML,omitempty" json:"captionHTML,omitempty"`
	Title       string `bson:"title,omitempty" json:"title,omitempty"`
}

// PixivMedia extends Media with extra info of Pixiv images, especially Ugoiras
type PixivMedia struct {
	UgoiraDelay []int `bson:"ugoiraDelay,omitempty" json:"ugoiraDelay,omitempty"`
}
