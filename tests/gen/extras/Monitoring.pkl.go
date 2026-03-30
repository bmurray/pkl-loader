// Code generated from Pkl module `test_extras.Monitoring`. DO NOT EDIT.
package extras

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

// Monitoring and alerting configuration.
type Monitoring struct {
	Endpoint string `pkl:"endpoint"`

	Interval uint16 `pkl:"interval"`

	AlertEmail string `pkl:"alertEmail"`

	Enabled bool `pkl:"enabled"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Monitoring
func LoadFromPath(ctx context.Context, path string) (ret Monitoring, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Monitoring
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Monitoring, error) {
	var ret Monitoring
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
