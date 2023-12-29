#!/usr/bin/env zsh
export NAMESPACE=cert-secret-syncer

case "$1" in
"install")
  helm install latest --namespace $NAMESPACE --create-namespace --timeout 10m -f ./values.yaml .
  ;;
"upgrade")
  helm upgrade latest --namespace $NAMESPACE --timeout 10m -f ./values.yaml .
  ;;
"uninstall")
  helm uninstall latest --namespace $NAMESPACE
  ;;
*)
  echo "Invalid option. Please choose install, upgrade, or uninstall."
  ;;
esac
