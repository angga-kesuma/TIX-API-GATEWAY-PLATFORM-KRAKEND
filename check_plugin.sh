#!/bin/sh
set -e

echo "[CHECK_PLUGIN] loading plugins..."

if [ ! -f /opt/krakend/plugins/middleware.so ] || [ ! -f /opt/krakend/plugins/http-server.so ]; then
  echo "[CHECK_PLUGIN] ERROR: Required plugins not found"
  exit 1
fi

echo "[CHECK_PLUGIN] testing middleware.so.."
krakend test-plugin -c /opt/krakend/plugins/middleware.so
echo "[CHECK_PLUGIN] testing http-server.so.."
krakend test-plugin -s /opt/krakend/plugins/http-server.so

echo "[CHECK_PLUGIN] plugins OK"

exit 0
