#!/usr/bin/env bash

# Modified from https://github.com/kubernetes-sigs/kueue/blob/065451d907fa27a0647436505b3cac38718ef136/hack/update-codegen.sh
# Apache-2.0, Copyright 2023 The Kubernetes Authors

set -o errexit
set -o nounset
set -o pipefail

GO_CMD=${1:-go}
PKG_ROOT=$(realpath "$(dirname ${BASH_SOURCE[0]})/..")
CODEGEN_PKG=$($GO_CMD list -m -f "{{.Dir}}" k8s.io/code-generator)

cd $PKG_ROOT

source "${CODEGEN_PKG}/kube_codegen.sh"

# TODO: remove the workaround when the issue is solved in code-generator
# (https://github.com/kubernetes/code-generator/issues/165).
# kube_codegen.sh expects this layout:
# .
# └── github.com
#     └── refat75
#         └── api-server-controller
# We can use soft links in order to fake this layout, such that
# ./github.com/refat75/codegen-demo resolves to ././../codegen-demo, or ./.

ln -s . github.com
ln -s .. refat75
trap "rm github.com && rm refat75" EXIT

kube::codegen::gen_helpers \
  --boilerplate /dev/null \
  "${PKG_ROOT}/pkg/apis"


kube::codegen::gen_client \
  --output-dir "${PKG_ROOT}/pkg/generated" \
  --output-pkg "github.com/refat75/api-server-controller/pkg/generated" \
  --boilerplate /dev/null \
  --with-watch \
  --with-applyconfig \
  "${PKG_ROOT}/pkg/apis"


# clean up temporary libraries added in go.mod by code-generator
"${GO_CMD}" mod tidy