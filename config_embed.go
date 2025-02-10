package what

import (
	"embed"
	_ "embed"
)

//go:embed config
var ConfigData embed.FS
