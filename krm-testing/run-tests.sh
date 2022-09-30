#!/usr/bin/env bash

set -euo pipefail

cd ./krm-cont-gen && ./run-it.sh && cd -
cd ./krm-cont-trans && ./run-it.sh && cd -
cd ./krm-exec-gen && ./run-it.sh && cd -
cd ./krm-exec-trans && ./run-it.sh && cd -
cd ./legacy-gen && ./run-it.sh && cd -
cd ./legacy-trans && ./run-it.sh && cd -

echo "TESTS PASSED"
