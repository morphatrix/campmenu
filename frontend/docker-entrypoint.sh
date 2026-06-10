#!/bin/sh
set -e

# Runs from the nginx image's /docker-entrypoint.d/ hook directory BEFORE nginx
# starts, so it must return (not exec nginx). Injects the runtime API URL into
# index.html (placeholder set in index.html), letting one built image be
# reconfigured via env/ConfigMap without rebuilding.
API_URL="${API_URL:-}"
INDEX=/usr/share/nginx/html/index.html

if [ -f "$INDEX" ]; then
  sed -i "s|__API_URL__|${API_URL}|g" "$INDEX"
fi
