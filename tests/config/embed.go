package config

import "embed"

//go:embed *.pkl PklProject PklProject.deps.json nested/*.pkl directnest/*.pkl
var FS embed.FS
