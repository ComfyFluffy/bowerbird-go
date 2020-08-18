package pixiv

// generated with /scripts/mongo_pipeline.py

var PipelinePostsAll = a{
	d{
		{Key: "$sort", Value: d{
			{Key: "_id", Value: -1},
		}},
	},
	d{
		{Key: "$lookup", Value: d{
			{Key: "from", Value: "tags"},
			{Key: "localField", Value: "tagIDs"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "tags"},
		}},
	},
	d{
		{Key: "$lookup", Value: d{
			{Key: "from", Value: "post_details"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "postID"},
			{Key: "as", Value: "latestPostDetail"},
		}},
	},
	d{
		{Key: "$set", Value: d{
			{Key: "latestPostDetail", Value: d{
				{Key: "$arrayElemAt", Value: a{
					"$latestPostDetail",
					-1,
				}},
			}},
		}},
	},
	d{
		{Key: "$lookup", Value: d{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "ownerID"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "owner"},
		}},
	},
	d{
		{Key: "$set", Value: d{
			{Key: "owner", Value: d{
				{Key: "$arrayElemAt", Value: a{
					"$owner",
					0,
				}},
			}},
		}},
	},
	d{
		{Key: "$lookup", Value: d{
			{Key: "from", Value: "media"},
			{Key: "localField", Value: "owner.avatarIDs"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "owner.avatar"},
		}},
	},
	d{
		{Key: "$set", Value: d{
			{Key: "owner.avatar", Value: d{
				{Key: "$arrayElemAt", Value: a{
					"$owner.avatar",
					-1,
				}},
			}},
		}},
	},
	d{
		{Key: "$lookup", Value: d{
			{Key: "from", Value: "media"},
			{Key: "localField", Value: "latestPostDetail.mediaIDs"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "latestPostDetail.media"},
		}},
	},
	d{
		{Key: "$unset", Value: a{
			"ownerID",
			"tagIDs",
			"latestPostDetail.postID",
			"owner.avatarIDs",
			"latestPostDetail.mediaIDs",
		}},
	},
}

var PipelineUsersAll = a{
	d{
		{Key: "$sort", Value: d{
			{Key: "_id", Value: -1},
		}},
	},
	d{
		{Key: "$lookup", Value: d{
			{Key: "from", Value: "media"},
			{Key: "localField", Value: "avatarIDs"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "avatar"},
		}},
	},
	d{
		{Key: "$set", Value: d{
			{Key: "avatar", Value: d{
				{Key: "$arrayElemAt", Value: a{
					"$avatar",
					-1,
				}},
			}},
		}},
	},
	d{
		{Key: "$lookup", Value: d{
			{Key: "from", Value: "user_details"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "userID"},
			{Key: "as", Value: "userDetail"},
		}},
	},
	d{
		{Key: "$set", Value: d{
			{Key: "userDetail", Value: d{
				{Key: "$arrayElemAt", Value: a{
					"$userDetail",
					-1,
				}},
			}},
		}},
	},
	d{
		{Key: "$unset", Value: a{
			"avatarIDs",
			"userDetail.userID",
		}},
	},
}
