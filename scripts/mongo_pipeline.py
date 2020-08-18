pipelinePostsAll = [{
    '$sort': {
        '_id': -1
    }
}, {
    '$lookup': {
        'from': 'tags',
        'localField': 'tagIDs',
        'foreignField': '_id',
        'as': 'tags'
    }
}, {
    '$lookup': {
        'from': 'post_details',
        'localField': '_id',
        'foreignField': 'postID',
        'as': 'latestPostDetail'
    }
}, {
    '$set': {
        'latestPostDetail': {
            '$arrayElemAt': ['$latestPostDetail', -1]
        }
    }
}, {
    '$lookup': {
        'from': 'users',
        'localField': 'ownerID',
        'foreignField': '_id',
        'as': 'owner'
    }
}, {
    '$set': {
        'owner': {
            '$arrayElemAt': ['$owner', 0]
        }
    }
}, {
    '$lookup': {
        'from': 'media',
        'localField': 'owner.avatarIDs',
        'foreignField': '_id',
        'as': 'owner.avatar'
    }
}, {
    '$set': {
        'owner.avatar': {
            '$arrayElemAt': ['$owner.avatar', -1]
        }
    }
}, {
    '$lookup': {
        'from': 'media',
        'localField': 'latestPostDetail.mediaIDs',
        'foreignField': '_id',
        'as': 'latestPostDetail.media'
    }
}, {
    '$unset': [
        'ownerID', 'tagIDs', 'latestPostDetail.postID', 'owner.avatarIDs',
        'latestPostDetail.mediaIDs'
    ]
}]

pipelineUsersAll = [
    {
        '$sort': {
            '_id': -1
        }
    }, {
        '$lookup': {
            'from': 'media', 
            'localField': 'avatarIDs', 
            'foreignField': '_id', 
            'as': 'avatar'
        }
    }, {
        '$set': {
            'avatar': {
                '$arrayElemAt': [
                    '$avatar', -1
                ]
            }
        }
    }, {
        '$lookup': {
            'from': 'user_details', 
            'localField': '_id', 
            'foreignField': 'userID', 
            'as': 'userDetail'
        }
    }, {
        '$set': {
            'userDetail': {
                '$arrayElemAt': [
                    '$userDetail', -1
                ]
            }
        }
    }, {
        '$unset': [
            'avatarIDs', 'userDetail.userID'
        ]
    }
]

# parse pipeline to golang bson.A
def parseObjToGo(a) -> str:
    if isinstance(a, dict):
        s = 'd{\n'
        for k, v in a.items():
            s += '{Key: %s, Value: %s},\n' % (parseObjToGo(k), parseObjToGo(v))
        s += '}'
        return s
    if isinstance(a, list):
        s = 'a{\n'
        for v in a:
            s += parseObjToGo(v)
            s += ',\n'
        s += '}'
        return s
    if isinstance(a, str):
        return '"%s"' % a.replace('"', '\\"')
    if isinstance(a, int):
        return str(a)
    if a == True:
        return 'true'
    if a == False:
        return 'false'


print(parseObjToGo(pipelineUsersAll))
