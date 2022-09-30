#!/usr/bin/env bash

set -euo pipefail

cd ../.. && docker build . -t quay.io/justinkuli/scratchpad:policy-generator && cd -
mv out.yaml out.yaml.expected

##### THE COMMAND ###########
kustomize build . --enable-alpha-plugins --mount type=bind,source=$(pwd),target=/tmp/,readonly > out.yaml
#############################

diff out.yaml.expected out.yaml
rm out.yaml.expected
