#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -e .env ]]; then
  echo '.env already exists; nothing changed.'
  exit 0
fi
umask 077
python3 - <<'PY'
from pathlib import Path
import secrets
source = Path('.env.example').read_text()
for key in ['POSTGRES_PASSWORD','REDIS_PASSWORD','JWT_SECRET','AUTHOR_INVITE_CODE']:
    source = source.replace(f'{key}=\n', f'{key}={secrets.token_hex(32)}\n')
with open('.env', 'x') as output:
    output.write(source)
print('Created .env with independent random secrets. Keep this file private.')
PY
