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

// PixivUserProfile extends UserProfile with Pixiv's user struct
type PixivUserProfile struct {
	Account        string            `bson:"account" json:"account"`
	Name           string            `bson:"name" json:"name"`
	IsPremium      bool              `bson:"isPremium" json:"isPremium"`
	Birth          string            `bson:"birth,omitempty" json:"birth,omitempty"`
	Country        string            `bson:"country,omitempty" json:"country,omitempty"`
	Gender         string            `bson:"gender,omitempty" json:"gender,omitempty"`
	TwitterAccount string            `bson:"twitterAccount,omitempty" json:"twitterAccount,omitempty"`
	WebPage        string            `bson:"webPage,omitempty" json:"webPage,omitempty"`
	Workspace      map[string]string `bson:"workspace,omitempty" json:"workspace,omitempty"`
}

// PixivIllust extends PostDetail with Pixiv's illust struct
type PixivIllust struct {
	IsBookmarked   bool `bson:"isBookmarked" json:"isBookmarked"`
	TotalBookmarks int  `bson:"totalBookmarks" json:"totalBookmarks"`
	TotalViews     int  `bson:"totalView" json:"totalView"`
}

// PixivIllustDetail extends PostDetail with Pixiv's illust struct
type PixivIllustDetail struct {
	// UgoiraDelay []int  `bson:"ugoiraDelay,omitempty" json:"ugoiraDelay,omitempty"`
	Type string `bson:"type,omitempty" json:"type"`
	// TODO
	Tools []string `bson:"tools,omitempty" json:"tools,omitempty"`
}

// PixivNovelDetail extends PostDetail with Pixiv's novel struct
type PixivNovelDetail struct {
	IsBookmarked   bool `bson:"isBookmarked" json:"isBookmarked"`
	TotalBookmarks int  `bson:"totalBookmarks" json:"totalBookmarks"`
	TotalViews     int  `bson:"totalView" json:"totalView"`
}

// PixivMedia extends Media with extra info of Pixiv images
type PixivMedia struct {
	UgoiraDelay []int `bson:"ugoiraDelay,omitempty" json:"ugoiraDelay,omitempty"`
}
