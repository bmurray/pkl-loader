package main

import (
	pklloader "github.com/bmurray/pkl-loader"
	"github.com/bmurray/pkl-loader/tests/config"
)

func main() {
	pklloader.RunExternalReader("testschema", config.FS)
}
