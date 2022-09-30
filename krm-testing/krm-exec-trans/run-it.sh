#!/usr/bin/env bash

set -euo pipefail

cd ../.. && make build-binary && cd -
mv out.yaml out.yaml.expected
cp ../../PolicyGenerator .

##### THE COMMAND ###########
kustomize build . --enable-alpha-plugins --enable-exec > out.yaml
#############################

diff out.yaml.expected out.yaml
rm out.yaml.expected
