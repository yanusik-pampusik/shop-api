#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
# Requires protoc, protoc-gen-go v1.36.12 and protoc-gen-go-grpc v1.6.2.
for pair in AuthService:authservice ShopAPI:shopapi; do
  directory=${pair%:*}
  module=${pair#*:}
  protoc -I proto/auth \
    --go_out="$directory" --go_opt="module=$module" \
    --go_opt="Mauth.proto=$module/proto/auth" \
    --go-grpc_out="$directory" --go-grpc_opt="module=$module" \
    --go-grpc_opt="Mauth.proto=$module/proto/auth" \
    auth.proto
done
