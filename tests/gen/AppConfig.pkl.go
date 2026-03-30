// Code generated from Pkl module `test_config.AppConfig`. DO NOT EDIT.
package gen

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"github.com/bmurray/pkl-loader/tests/gen/directnest"
	"github.com/bmurray/pkl-loader/tests/gen/nested"
	"github.com/bmurray/pkl-loader/tests/gen/subconfig"
)

// Top-level application configuration schema.
type AppConfig struct {
	AppName any `pkl:"appName"`

	Database Database `pkl:"database"`

	Features Features `pkl:"features"`

	Sub subconfig.Sub `pkl:"sub"`

	Nested nested.Nested `pkl:"nested"`

	Direct directnest.DirectConfig `pkl:"direct"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a AppConfig
func LoadFromPath(ctx context.Context, path string) (ret AppConfig, err error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return ret, err
	}
	defer func() {
		cerr := evaluator.Close()
		if err == nil {
			err = cerr
		}
	}()
	ret, err = Load(ctx, evaluator, pkl.FileSource(path))
	return ret, err
}

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a AppConfig
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (AppConfig, error) {
	var ret AppConfig
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
