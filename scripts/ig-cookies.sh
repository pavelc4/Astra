#!/bin/bash
set -euo pipefail

echo "=== Instagram Cookie Generator ==="
echo ""
echo "Masukkan cookie dari browser (Chrome F12 → Application → Cookies → instagram.com)"
echo ""

read -rp "sessionid: " sessionid
read -rp "csrftoken: " csrftoken

if [[ -z "$sessionid" || -z "$csrftoken" ]]; then
  echo "Error: sessionid dan csrftoken wajib diisi"
  exit 1
fi

cookie="sessionid=${sessionid}; csrftoken=${csrftoken}"

echo ""
echo "=== Copy baris di bawah ke .env ==="
echo "INSTAGRAM_COOKIES=\"${cookie}\""
