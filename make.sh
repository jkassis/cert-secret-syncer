#!/usr/bin/env zsh
export NAMESPACE=cert-secret-syncer

case "$1" in
"tidy")
  go mod tidy
  go mod vendor
  ;;
"buildpush")
  go run build/main.go build --VERSION latest
  export REGION=us-west-2
  go run build/main.go push
  ;;
"build")
  go run build/main.go build --VERSION latest
  ;;
"push")
  export REGION=us-west-2
  go run build/main.go push
  ;;
"install")
  cd helm
  helm install latest --namespace $NAMESPACE --create-namespace --timeout 10m -f ./values.yaml .
  ;;
"upgrade")
  cd helm
  helm upgrade latest --namespace $NAMESPACE --timeout 10m -f ./values.yaml .
  ;;
"uninstall")
  cd helm
  helm uninstall latest --namespace $NAMESPACE
  ;;
*)
  echo "Invalid option. Please choose install, upgrade, or uninstall."
  ;;
esac
