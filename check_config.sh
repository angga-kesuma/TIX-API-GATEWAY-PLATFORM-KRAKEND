#!/bin/sh
docker run --rm -v "${PWD}/example/config:/etc/krakend" -e FC_SETTINGS="/etc/krakend/settings" -e FC_ENABLE=1 -e FC_TEMPLATES="/etc/krakend/templates" -e FC_OUT=out.json krakend:2.10.2 krakend check -dc /etc/krakend/krakend.json
echo "Full configuration:"
cat "${PWD}/example/config/out.json"
