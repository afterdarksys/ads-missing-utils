#!/usr/bin/env python3
"""Demonstrate accessible context, event review, interruption recovery and handoff."""
import argparse
import datetime
import json
from pathlib import Path
import subprocess
import tempfile

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--rcdo', default=str(root.parent / 'rcdo/dist/rcdo'))
parser.add_argument('--bin-dir', default=str(root / 'dist'))
parser.add_argument('--output', help='new directory for evidence; default retained temporary directory')
args = parser.parse_args()
rcdo = str(Path(args.rcdo).resolve())
binaries = Path(args.bin_dir).resolve()
for path in (Path(rcdo), binaries / 'contextsnap', binaries / 'eventwhy'):
    if not path.is_file():
        raise SystemExit(f'Build the binary first: {path}')
out = Path(args.output).resolve() if args.output else Path(tempfile.mkdtemp(prefix='accessible-ops-demo-'))
if args.output:
    out.mkdir(parents=True, exist_ok=False)
out.chmod(0o700)
results = []

def write(name, value):
    path = out / name
    path.write_text(json.dumps(value, indent=2) + '\n')
    path.chmod(0o600)
    return path

def run(name, binary, arguments, expected):
    result = subprocess.run([str(binary), *map(str, arguments)], text=True, capture_output=True, timeout=30)
    path = out / name
    path.write_text(result.stdout)
    path.chmod(0o600)
    results.append({'step': name, 'expected_exit': expected, 'actual_exit': result.returncode})
    if result.returncode != expected:
        raise SystemExit(f'{name}: expected {expected}, got {result.returncode}\n{result.stderr}\n{result.stdout}')
    return path

config = out / 'config.yaml'
run('configuration.log', rcdo, ['config', 'init', '--file', config], 0)
def review(name, command, arguments, expected):
    return run(name, rcdo, [command, '--config-file', config, *arguments], expected)

print('LOCAL FIXTURE DEMO: no live cloud identity or production operation is claimed.')
at = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
for cloud, region, key in [('aws', 'us-east-1', 'Account'), ('alicloud', 'cn-hangzhou', 'AccountId')]:
    expected = write(f'{cloud}-expected.json', {'schema_version': '1', 'name': f'demo-{cloud}', 'environment': 'demo', 'cloud': cloud, 'profile': 'demo', 'region': region, 'account': '123456789012'})
    source = write(f'{cloud}-identity.json', {key: '123456789012', 'Arn': 'demo-principal'})
    snapshot = run(f'{cloud}-snapshot.json', binaries / 'contextsnap', ['--cloud', cloud, '--profile', 'demo', '--region', region, '--input', source, '--observed-at', at], 0)
    review(f'{cloud}-context.json', 'context', ['--input', snapshot, '--expect', expected, '--format', 'json'], 0)
    review(f'{cloud}-context.txt', 'context', ['--input', snapshot, '--expect', expected, '--width', '72'], 0)
    wrong = write(f'{cloud}-wrong-expected.json', {**json.loads(expected.read_text()), 'account': '999999999999'})
    review(f'{cloud}-wrong.txt', 'context', ['--input', snapshot, '--expect', wrong, '--width', '72'], 20)
    print(f'{cloud}: matching context accepted; wrong account BLOCKED.')

now = int(datetime.datetime.now(datetime.timezone.utc).timestamp())
events = out / 'docker-events.ndjson'
events.write_text('\n'.join(json.dumps({'Type': 'container', 'Action': action, 'time': now+i, 'Actor': {'ID': resource, 'Attributes': {'exitCode': '137', 'private-label': 'not-retained'}}}) for i, (resource, action) in enumerate([('demo-api', 'die')] * 3 + [('demo-worker', 'oom')])) + '\n')
events.chmod(0o600)
groups = run('events.json', binaries / 'eventwhy', ['--input', events, '--context', 'fixture-only'], 0)
parsed = json.loads(groups.read_text())
assert parsed['events'] == 4 and len(parsed['groups']) == 2
assert 'private-label' not in groups.read_text()
report = review('events-review.json', 'watch', ['--input', groups, '--environment', 'demo', '--format', 'json'], 20)
review('events-review.txt', 'watch', ['--input', groups, '--environment', 'demo', '--width', '72'], 20)
print('Four Docker events grouped into two observations; raw labels omitted.')

session = out / 'incident-session.json'
review('start.txt', 'review-session', ['start', '--report', report, '--session', session, '--change-id', 'DEMO-INCIDENT', '--commit', 'fixture-only', '--artifact', groups, '--owner', 'Demo engineer', '--impact', 'Demo API is unavailable; scenario statement.', '--next-action', 'Inspect current health before selecting remediation.', '--width', '72'], 0)
review('forward.txt', 'review-session', ['forward', '--session', session, '--width', '72'], 20)
review('bookmark.txt', 'review-session', ['bookmark', '--session', session, '--name', 'after-interruption', '--width', '72'], 0)
review('note.txt', 'review-session', ['note', '--session', session, '--note', 'Exit 137 alone does not prove OOM; inspect memory evidence.', '--width', '72'], 0)
resumed = review('resume.txt', 'review-session', ['resume', '--session', session, '--width', '72'], 20)
assert 'after-interruption' in resumed.read_text() and 'Exit 137 alone' in resumed.read_text()
print('Interrupted review resumed with the same finding, bookmark, and note.')
for item in json.loads(report.read_text())['findings']:
    review('ack.txt', 'review-session', ['ack', '--session', session, '--id', item['id'], '--note', 'Read in demo; issue remains unresolved.', '--width', '72'], 0)
handoff = review('handoff.txt', 'handoff', ['--session', session, '--width', '72'], 20)
assert 'not resolved' in handoff.read_text()
print('Handoff preserves unresolved blockers after every finding is acknowledged.')
with groups.open('a') as stream:
    stream.write('\n')
history = review('stale-resume.txt', 'review-session', ['resume', '--session', session, '--width', '72'], 30)
assert 'HISTORICAL EVIDENCE' in history.read_text() and 'after-interruption' in history.read_text()
review('stale-handoff.txt', 'handoff', ['--session', session, '--width', '72'], 30)
print('Changed evidence: INCOMPLETE, with reading position and notes preserved.')
for path in out.glob('*.txt'):
    review('output-check.log', 'a11y-output-check', ['--input', path, '--max-line', '72'], 0)
write('acceptance-results.json', results)
review('evidence.json', 'evidence-pack', ['--change-id', 'DEMO-INCIDENT', '--commit', 'fixture-only', '--file', session, '--file', report, '--file', handoff, '--file', out / 'acceptance-results.json'], 0)
print('All command outcomes and 72-column output checks passed.')
print(f'Reviewable evidence: {out}')
