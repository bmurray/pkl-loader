package fixtures

import "embed"

//go:embed all:*.pkl all:**/*.pkl PklProject PklProject.deps.json
var FS embed.FS
