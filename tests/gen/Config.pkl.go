// Code generated from Pkl module `test_config.AppConfig`. DO NOT EDIT.
package gen

import (
	"github.com/bmurray/pkl-loader/tests/gen/nested"
	"github.com/bmurray/pkl-loader/tests/gen/subconfig"
)

// Top-level application configuration.
type Config struct {
	AppName string `pkl:"appName"`

	Database Database `pkl:"database"`

	Features Features `pkl:"features"`

	Sub subconfig.Sub `pkl:"sub"`

	Nested nested.Nested `pkl:"nested"`
}
