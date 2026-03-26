#!/bin/bash
#
# move-media.sh
#
# Changes the group of encoded files and moves them
# to a destination media directory.
#
# HandyMKV automation parameters (environment variables):
#   HMKV_PARAM_ENCODED_DIR   - Source directory containing encoded files (hmkv_output: hb_output_dir)
#   HMKV_PARAM_MEDIA_DIR     - Destination media directory (source: prompt)
#   HMKV_PARAM_GROUP_NAME    - Group to assign to files (source: static)
#

set -euo pipefail

if [ -z "${HMKV_PARAM_ENCODED_DIR:-}" ]; then
    echo "Error: HMKV_PARAM_ENCODED_DIR is not set."
    exit 1
fi

if [ -z "${HMKV_PARAM_MEDIA_DIR:-}" ]; then
    echo "Error: HMKV_PARAM_MEDIA_DIR is not set."
    exit 1
fi

if [ -z "${HMKV_PARAM_GROUP_NAME:-}" ]; then
    echo "Error: HMKV_PARAM_GROUP_NAME is not set."
    exit 1
fi

if [ ! -d "$HMKV_PARAM_ENCODED_DIR" ]; then
    echo "Error: Encoded directory does not exist: $HMKV_PARAM_ENCODED_DIR"
    exit 1
fi

mkdir -p "$HMKV_PARAM_MEDIA_DIR"

find "$HMKV_PARAM_ENCODED_DIR" -type f | while read -r file; do
    chgrp "$HMKV_PARAM_GROUP_NAME" "$file"
    chmod 0774 "$file"
    mv "$file" "$HMKV_PARAM_MEDIA_DIR/"
    echo "Moved: $(basename "$file")"
done

echo "Done. Files moved to $HMKV_PARAM_MEDIA_DIR"
