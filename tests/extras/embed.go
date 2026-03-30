package extras

import "embed"

//go:embed *.pkl PklProject PklProject.deps.json
var FS embed.FS
