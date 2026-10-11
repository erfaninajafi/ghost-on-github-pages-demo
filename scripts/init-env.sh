#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
umask 077
python3 - <<'PYENV'
from pathlib import Path
import secrets
path = Path('.env')
if path.exists():
    source = path.read_text()
    keys = {line.split('=', 1)[0].strip(): line.split('=', 1)[1].strip() for line in source.splitlines() if '=' in line and not line.lstrip().startswith('#')}
    if not keys.get('REVALIDATION_SECRET'):
        with path.open('a') as output:
            output.write('\nREVALIDATION_SECRET=' + secrets.token_hex(32) + '\n')
        print('Added REVALIDATION_SECRET; existing settings unchanged.')
    else:
        print('.env already exists; nothing changed.')
else:
    source = Path('.env.example').read_text()
    for key in ['POSTGRES_PASSWORD', 'REDIS_PASSWORD', 'JWT_SECRET', 'REVALIDATION_SECRET']:
        source = source.replace(f'{key}=\n', f'{key}={secrets.token_hex(32)}\n')
    with path.open('x') as output:
        output.write(source)
    print('Created .env with independent random secrets. Keep this file private.')
PYENV
