package downloader

import "strings"

// Replace special characters with fullwidth characters

var replacerAll = strings.NewReplacer(
	"/", "／",
)

var replacerOnWindows = strings.NewReplacer(
	":", "：",
	"*", "＊",
	"?", "？",
	"\"", "“",
	"<", "＜",
	">", "＞",
	"|", "｜",
)
