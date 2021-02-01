set -x
set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT=$(cd $(dirname ${BASH_SOURCE})/..; pwd)

export GO111MODULE=off

echo "Building protobuf-gen"
go build -i -o dist/go-to-protobuf ./vendor/k8s.io/code-generator/cmd/go-to-protobuf

PACKAGES=(
    github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1
)

APIMACHINERY_PKGS=(
    k8s.io/apimachinery/pkg/apis/meta/v1
    k8s.io/api/core/v1
)

"${PROJECT_ROOT}"/dist/go-to-protobuf \
    --go-header-file="${PROJECT_ROOT}"/hack/boilerplate/boilerplate.generatego.txt \
    --packages=$(IFS=, ; echo "${PACKAGES[*]}") \
    --apimachinery-packages=$(IFS=, ; echo "${APIMACHINERY_PKGS[*]}") \
    --proto-import=/usr/local/include/google/protobuf/ \
    -o="${HOME}"/go/src/


protoc -I "${HOME}"/go/src \
      -I "${PROJECT_ROOT}"/vendor \
      -I pkg/comm/controller/pb \
      pkg/comm/controller/pb/controller.proto \
      --go_out=plugins=grpc:pkg/comm/controller/pb/

protoc -I "${HOME}"/go/src \
      -I "${PROJECT_ROOT}"/vendor \
      -I pkg/comm/bundle/pb \
      pkg/comm/bundle/pb/bundle.proto \
      --go_out=plugins=grpc:pkg/comm/bundle/pb/