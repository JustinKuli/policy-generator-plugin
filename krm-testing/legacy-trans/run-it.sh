#!/usr/bin/env bash

set -euo pipefail

cd ../.. && make build && cd -
mv out.yaml out.yaml.expected

##### THE COMMAND ###########
kustomize build . --enable-alpha-plugins > out.yaml
#############################

diff out.yaml.expected out.yaml
rm out.yaml.expected
