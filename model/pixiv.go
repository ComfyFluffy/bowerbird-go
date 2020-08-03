package model

// PixivUser extends User with Pixiv's user struct
type PixivUser struct {
	IsFollowed           bool `bson:"isFollowed" json:"isFollowed"`
	TotalFollowing       int  `bson:"totalFollowing" json:"totalFollowing"`
	TotalPublicBookmarks int  `bson:"totalPublicBookmarks" json:"totalPublicBookmarks"`
	TotalIllusts         int  `bson:"totalIllusts" json:"totalIllusts"`
	TotalManga           int  `bson:"totalManga" json:"totalManga"`
	TotalNovels          int  `bson:"totalNovels" json:"totalNovels"`
	TotalIllustSeries    int  `bson:"totalIllustSeries" json:"totalIllustSeries"`
	TotalNovelSeries     int  `bson:"totalNovelSeries" json:"totalNovelSeries"`
}

// PixivUserProfile extends UserDetail with Pixiv's user struct
type PixivUserProfile struct {
	Account        string `bson:"account" json:"account"`
	IsPremium      bool   `bson:"isPremium" json:"isPremium"`
	Birth          string `bson:"birth,omitempty" json:"birth,omitempty"`
	Region         string `bson:"region,omitempty" json:"region,omitempty"`
	Gender         string `bson:"gender,omitempty" json:"gender,omitempty"`
	TwitterAccount string `bson:"twitterAccount,omitempty" json:"twitterAccount,omitempty"`
	WebPage        string `bson:"webPage,omitempty" json:"webPage,omitempty"`
	Bio            string `bson:"bio,omitempty" json:"bio,omitempty"`
	Workspace      DD     `bson:"workspace,omitempty" json:"workspace,omitempty"`
	WorkspaceImage *Media `bson:"workspaceImage,omitempty" json:"workspaceImage,omitempty"`
}

// PixivPost extends PostDetail with Pixiv's illust struct
type PixivPost struct {
	IsBookmarked   bool `bson:"isBookmarked" json:"isBookmarked"`
	TotalBookmarks int  `bson:"totalBookmarks" json:"totalBookmarks"`
	TotalViews     int  `bson:"totalView" json:"totalView"`
}

// PixivIllustDetail extends PostDetail with Pixiv's illust struct
type PixivIllustDetail struct {
	// Type can be "illust", "manga" or "novel"
	Type string `bson:"type,omitempty" json:"type"`
}

// PixivMedia extends Media with extra info of Pixiv images, especially Ugoiras
type PixivMedia struct {
	UgoiraDelay []int `bson:"ugoiraDelay,omitempty" json:"ugoiraDelay,omitempty"`
}
