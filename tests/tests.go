package tests

//go:generate rm -rf gen
//go:generate sh -c "cd .. && pkl run 'package://pkg.pkl-lang.org/pkl-go/pkl.golang@0.13.2#/gen.pkl' -p projectDir=tests/config -p moduleDir=tests/config --output-path . -- tests/config/Config.pkl tests/config/SubConfig.pkl tests/config/nested/NestedConfig.pkl tests/config/directnest/DirectConfig.pkl"
//go:generate sh -c "cd .. && pkl run 'package://pkg.pkl-lang.org/pkl-go/pkl.golang@0.13.2#/gen.pkl' -p projectDir=tests/extras -p moduleDir=tests/extras --output-path . -- tests/extras/Monitoring.pkl"
//go:generate sh -c "cd .. && pkl run 'package://pkg.pkl-lang.org/pkl-go/pkl.golang@0.13.2#/gen.pkl' --project-dir tests/shorthand -p projectDir=tests/shorthand -p moduleDir=tests/shorthand --output-path . -- tests/shorthand/Config.pkl"
