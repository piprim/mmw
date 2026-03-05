#!/usr/bin/env bash
set -euo pipefail

D_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$D_SCRIPT_DIR/libs/st.bash"

st.quiet

st.h1 "== mise =="
st.do mise --version || true

echo
st.h1 "== go =="
st.do go version
echo "GOBIN=$(go env GOBIN)"
echo "GOPATH=$(go env GOPATH)"

echo
st.h1 "== gopls =="
command -v gopls >/dev/null && gopls version || st.warn "gopls: not found (run: mise run tools)"

echo
st.h1 "== golangci-lint-langserver =="
command -v golangci-lint-langserver || st.warn "golangci-lint-langserver not found (run: mise run tools)"

echo
st.h1 "== golangci-lint =="
command -v golangci-lint >/dev/null && golangci-lint --version || st.warn "golangci-lint: not found (run: mise install)"

echo
st.h1 "== buf =="
command -v buf >/dev/null && buf --version || st.warn "buf: not found (run: mise install)"

echo
st.h1 "== protoc =="
command -v protoc >/dev/null && protoc --version || st.warn "protoc: not found (apt install protobuf-compiler) or use devbox"

echo
st.h1 "== protoc-gen-go =="
command -v protoc-gen-go >/dev/null && protoc-gen-go --version || st.warn "protoc-gen-go: not found (run: mise run tools)"

echo
st.h1 "== protoc-gen-go-grpc =="
command -v protoc-gen-go-grpc >/dev/null && protoc-gen-go-grpc --version || st.warn "protoc-gen-go-grpc: not found (run: mise run tools)"
