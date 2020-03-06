package model

// PixivUser extends User with Pixiv's user struct
type PixivUser struct {
	Name                 string `json:"name"`
	Account              string `json:"account"`
	IsFollowed           bool   `json:"isFollowed"`
	IsPremium            bool   `json:"isPremium"`
	TotalFollowing       int    `json:"totalFollowing"`
	TotalPublicBookmarks int    `json:"totalPublicBookmarks"`
	TotalIllusts         int    `json:"totalIllusts"`
	TotalManga           int    `json:"totalManga"`
	TotalNovels          int    `json:"totalNovels"`
	TotalIllustSeries    int    `json:"totalIllustSeries"`
	TotalNovelSeries     int    `json:"totalNovelSeries"`

	Birth          string            `bson:",omitempty" json:"birth,omitempty"`
	Country        string            `bson:",omitempty" json:"country,omitempty"`
	Gender         string            `bson:",omitempty" json:"gender,omitempty"`
	TwitterAccount string            `bson:",omitempty" json:"twitterAccount,omitempty"`
	WebPage        string            `bson:",omitempty" json:"webPage,omitempty"`
	Workspace      map[string]string `bson:",omitempty" json:"workspace,omitempty"`
}

// PixivIllust extends PostDetail with Pixiv's illust struct
type PixivIllust struct {
	Likes          int   `json:"likes"`
	IsBookmarked   bool  `json:"isBookmarked"`
	TotalBookmarks int   `json:"totalBookmarks"`
	TotalView      int   `json:"totalView"`
	UgoiraDelay    []int `json:"ugoiraDelay,omitempty"`
	// TODO
	Tools []string
}

type PixivNovel struct {
}
