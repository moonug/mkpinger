#!/usr/bin/env sh
set -e

if [ -z "$1" ]; then
       		exec /pinger $ARGS
fi

exec "$@"
