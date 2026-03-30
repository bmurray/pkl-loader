// Code generated from Pkl module `test_config.SubConfig`. DO NOT EDIT.
package subconfig

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

// Configuration imported from the same directory.
type SubConfig struct {
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a SubConfig
func LoadFromPath(ctx context.Context, path string) (ret SubConfig, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a SubConfig
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (SubConfig, error) {
	var ret SubConfig
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
