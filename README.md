# Bowerbird

A toolset for downloading & managing items from websites.

Type `bowerbird --help` for full CLI usage.

Supported websites:

- pixiv

## Database

This tool uses [MongoDB](https://www.mongodb.com/) as database.

## Websites

### pixiv

The tool can download illustrations/manga/ugoira(zip) from pixiv.net and save their information to database.

- Save user's own bookmarks:

  `bowerbird pixiv bookmark`

  - Save to database but not download them:

    `bowerbird --db-only pixiv bookmark`
  
  - Download them without database:

    `bowerbird --no-db pixiv bookmark`

  - With a limit of 30 works:

    `bowerbird pixiv --limit 30 bookmark`

  - Match any of the given tags

    `bowerbird pixiv -t "風景" -t "男の子" bookmark`
  - Match all tags:

    `bowerbird pixiv -t "風景" -t "男の子" --tags-match-all bookmark`

- Save user's works with pixiv user ID:

  `bowerbird pixiv -u 4177162 uploads`
