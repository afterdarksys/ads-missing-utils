#!/usr/bin/env python3
"""Render the README command matrix from reviewed command metadata."""
import argparse
import json
import textwrap
from pathlib import Path

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--check', action='store_true', help='fail if metadata or generated documentation is stale')
args = parser.parse_args()
commands = json.loads((root / 'docs/commands.json').read_text())
names = [item['name'] for item in commands]
actual = sorted(path.name for path in (root / 'cmd').iterdir() if path.is_dir())
if sorted(names) != actual or len(set(names)) != len(names):
    raise SystemExit('Command metadata must cover every cmd directory exactly once.')
lines = ['<!-- BEGIN GENERATED CAPABILITIES -->',
         'This matrix describes implemented scope, not production certification. All commands are unreleased.', '',
         '| Command | Scope | Boundary |', '|---|---|---|']
for item in sorted(commands, key=lambda value: value['name']):
    if item['status'] != 'implemented-unreleased':
        raise SystemExit('Review the renderer before introducing another release status.')
    lines.append(f"| `{item['name']}` | {item['scope']} | {item['boundary']} |")
lines.append('<!-- END GENERATED CAPABILITIES -->')
readme = root / 'README.md'
text = readme.read_text()
start = text.index('<!-- BEGIN GENERATED CAPABILITIES -->')
end = text.index('<!-- END GENERATED CAPABILITIES -->') + len('<!-- END GENERATED CAPABILITIES -->')
updated = text[:start] + '\n'.join(lines) + text[end:]
linear = ['Missing Utils command capabilities', 'Status: implemented, unreleased; bounded scope, not production certification.', '']
for item in sorted(commands, key=lambda value: value['name']):
    for label, value in [('Command', item['name']), ('Scope', item['scope']), ('Boundary', item['boundary'])]:
        linear.extend(textwrap.wrap(f'{label}: {value}', width=72))
    linear.append('')
linear_text = '\n'.join(linear)
linear_path = root / 'docs/COMMANDS.txt'
if args.check:
    if updated != text or not linear_path.exists() or linear_path.read_text() != linear_text:
        raise SystemExit('Capability matrix is stale. Run python3 scripts/update-capabilities.py.')
    print(f'Capability metadata and README agree for {len(names)} commands.')
else:
    readme.write_text(updated)
    linear_path.write_text(linear_text)
