#!/usr/bin/env python3
"""Exercise local readiness, change review, acknowledgement, and stale evidence."""
import argparse
import json
from pathlib import Path
import subprocess
import tempfile

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--rcdo', default=str(root.parent / 'rcdo/dist/rcdo'))
parser.add_argument('--jsonprobe', default=str(root / 'dist/jsonprobe'))
parser.add_argument('--output', help='new directory for the evidence; defaults to a temporary directory')
args = parser.parse_args()
rcdo, probe = str(Path(args.rcdo).resolve()), str(Path(args.jsonprobe).resolve())
for binary in (rcdo, probe):
    if not Path(binary).is_file():
        raise SystemExit(f'Build the binary first: {binary}')
output = Path(args.output).resolve() if args.output else Path(tempfile.mkdtemp(prefix='rcdo-demo-'))
if args.output:
    output.mkdir(parents=True, exist_ok=False)

def write(name, value):
    path = output / name
    path.write_text(json.dumps(value, indent=2) + '\n')
    path.chmod(0o600)
    return path

def run(name, binary, arguments, expected):
    result = subprocess.run([binary, *map(str, arguments)], text=True, capture_output=True, timeout=30)
    path = output / name
    path.write_text(result.stdout)
    path.chmod(0o600)
    if result.returncode != expected:
        raise SystemExit(f'{name}: expected exit {expected}, got {result.returncode}\n{result.stderr}')
    return path

# Isolate the demo from personal policy/default configuration.
config = output / 'config.yaml'
run('config-init.log', rcdo, ['config', 'init', '--file', config], 0)
def review(name, command, arguments, expected):
    return run(name, rcdo, [command, '--config-file', config, *arguments], expected)

marker = output / 'service-ready'
marker.write_text('local fixture only\n')
spec = write('probes.json', {'checks': [{'name': 'demo-ready', 'type': 'file', 'path': str(marker)}]})
raw = run('probes-result.json', probe, ['--input', spec], 0)
readiness = review('readiness.json', 'jsonprobe-check', ['--input', raw, '--require', 'demo-ready', '--environment', 'demo', '--format', 'json'], 0)
review('missing.json', 'jsonprobe-check', ['--input', raw, '--require', 'database-ready', '--environment', 'demo', '--format', 'json'], 30)
print('Missing required readiness check: INCOMPLETE (exit 30).')

for scenario, expected in [('safe', 0), ('destructive', 20)]:
    plan = write(f'{scenario}-plan.json', json.loads((root / f'examples/rcdo/{scenario}-plan.json').read_text()))
    plan_report = review(f'{scenario}-tofu.json', 'tofu-check', ['--input', plan, '--environment', 'demo', '--format', 'json'], expected)
    manifest = write(f'{scenario}-change.json', {
        'schema_version': '1', 'change_id': f'DEMO-{scenario}', 'commit': 'fixture-only',
        'environment': 'demo', 'required_components': ['opentofu', 'readiness'],
        'reports': {'opentofu': plan_report.name, 'readiness': readiness.name},
    })
    report = review(f'{scenario}-review.json', 'review-change', ['--manifest', manifest, '--format', 'json'], expected)
    review(f'{scenario}-brief.txt', 'review-brief', ['--input', report, '--width', '72'], expected)
    print(f'{scenario.capitalize()} fixture: {"CLEAN" if expected == 0 else "BLOCKED"} (exit {expected}).')

session = output / 'review-session.json'
review('session-start.txt', 'review-session', ['start', '--session', session, '--report', report, '--change-id', 'DEMO-destructive', '--commit', 'fixture-only', '--artifact', plan, '--artifact', raw, '--width', '72'], 0)
for item in json.loads(report.read_text())['findings']:
    review('session-ack.txt', 'review-session', ['ack', '--session', session, '--id', item['id'], '--note', 'Read in the local demo; deletion remains blocked.', '--width', '72'], 0)
review('session-read.txt', 'review-session', ['next', '--session', session, '--width', '72'], 20)
print('All findings acknowledged: still BLOCKED (exit 20).')
with plan.open('a') as stream:
    stream.write('\n')
review('session-stale.txt', 'review-session', ['next', '--session', session, '--width', '72'], 30)
print('Bound plan changed: session INCOMPLETE (exit 30).')
for path in output.glob('*.txt'):
    review('accessibility-check.json', 'a11y-output-check', ['--input', path, '--max-line', '72'], 0)
review('evidence.json', 'evidence-pack', ['--commit', 'fixture-only', '--change-id', 'DEMO', '--file', report, '--file', readiness, '--file', raw, '--file', session], 0)
print('Accessible text checks passed. No deployment commands were executed.')
print(f'Evidence directory: {output}')
