package downloader

import "strings"

// Replace special characters in path with fullwidth characters

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
	"\\", "／",
)
