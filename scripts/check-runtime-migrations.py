#!/usr/bin/env python3
"""Verify that application binaries exclude migration SQL and the Goose library."""
import argparse
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--binary-dir', type=Path, required=True)
parser.add_argument('--sql-dir', type=Path, required=True)
args = parser.parse_args()
sql = {p.name: p.read_bytes() for p in sorted(args.sql_dir.glob('*.sql'))}
if not sql:
    raise SystemExit('No versioned migration SQL found')

def binary(role):
    path = args.binary_dir / ('gin-enterprise-template-' + role)
    if not path.exists():
        path = path.with_suffix('.exe')
    return path.read_bytes()

failures = []
for role in ['apiserver', 'worker', 'scheduler']:
    data = binary(role)
    included = [name for name, content in sql.items() if content in data]
    if included:
        failures.append(role + ' embeds migration SQL: ' + ', '.join(included))
    else:
        print('PASS', role, 'does not embed migration SQL')
import subprocess
for role in ['apiserver', 'worker', 'scheduler']:
    path = args.binary_dir / ('gin-enterprise-template-' + role)
    if not path.exists():
        path = path.with_suffix('.exe')
    metadata = subprocess.run(['go', 'version', '-m', str(path)], capture_output=True, text=True, check=True).stdout
    includes_goose = 'github.com/pressly/goose/v3' in metadata
    if includes_goose:
        failures.append(role + ' has unexpected Goose dependency')
    else:
        print('PASS', role, 'Goose dependency isolation')
if failures:
    raise SystemExit('\n'.join(failures))
