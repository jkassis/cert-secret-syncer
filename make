#!/usr/bin/env zsh
version="${2:-latest}"

export HOST=docker.io
export IMAGE=jkassis/cert-secret-syncer
export VERSION="$version"

case "$1" in
"tidy")
  echo "Tidying and vendoring Go modules..."
  go mod tidy || exit 1
  go mod vendor || exit 1
  ;;

"buildpush")
  platforms="${PLATFORMS:-linux/amd64,linux/arm64}"
  echo "Building and pushing multi-arch ($platforms) with VERSION: $tag..."
  PLATFORMS="$platforms" PUSH=true go run build/main.go build || exit 1
  ;;

"build")
  echo "Building with VERSION: $tag..."
  go run build/main.go build || exit 1
  ;;

"push")
  echo "Pushing latest build..."
  go run build/main.go push || exit 1
  ;;

"help" | "-h")
  echo "Usage: $0 [tidy|buildpush|build|push] [VERSION]"
  ;;

*)
  echo "Invalid option. Use 'help' or '-h' for usage instructions."
  ;;
esac
