#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."

# protoc 33.0; protoc-gen-go v1.36.12; protoc-gen-go-grpc v1.6.2.
protoc -I ../proto/auth \
  --go_out=. --go_opt=module=authservice \
  --go_opt=Mauth.proto=authservice/proto/auth \
  --go-grpc_out=. --go-grpc_opt=module=authservice \
  --go-grpc_opt=Mauth.proto=authservice/proto/auth \
  auth.proto
