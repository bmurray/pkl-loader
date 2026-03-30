// Code generated from Pkl module `test_config.directnest.DirectConfig`. DO NOT EDIT.
package directnest

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

// Flat configuration with no sub-classes and sensible defaults.
type DirectConfig struct {
	AppName string `pkl:"appName"`

	Host string `pkl:"host"`

	Port uint16 `pkl:"port"`

	EnableCache bool `pkl:"enableCache"`

	MaxRetries uint8 `pkl:"maxRetries"`

	Region string `pkl:"region"`

	Label string `pkl:"label"`

	Priority uint8 `pkl:"priority"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a DirectConfig
func LoadFromPath(ctx context.Context, path string) (ret DirectConfig, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a DirectConfig
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (DirectConfig, error) {
	var ret DirectConfig
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
