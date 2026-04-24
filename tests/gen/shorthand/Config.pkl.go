// Code generated from Pkl module `shorthand_config.Config`. DO NOT EDIT.
package shorthand

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

// Simple config using shorthand dependency import.
type Config struct {
	AppName string `pkl:"appName"`

	Port uint16 `pkl:"port"`

	Debug bool `pkl:"debug"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Config
func LoadFromPath(ctx context.Context, path string) (ret Config, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Config
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Config, error) {
	var ret Config
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
