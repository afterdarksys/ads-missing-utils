#!/usr/bin/env python3
"""Offline cross-tool acceptance: pipeline failure, navigation, stale edit."""
import argparse, hashlib, json, pathlib, subprocess, sys, tempfile
p=argparse.ArgumentParser(description=__doc__)
p.add_argument('--rcdo',default='../rcdo/dist/rcdo')
p.add_argument('--runreceipt',default='dist/runreceipt')
a=p.parse_args()
rcdo=str(pathlib.Path(a.rcdo).resolve()); receipt=str(pathlib.Path(a.runreceipt).resolve())
def run(args,code=0):
    r=subprocess.run(args,text=True,capture_output=True)
    assert r.returncode==code,(args,r.returncode,r.stdout,r.stderr)
    return r.stdout
with tempfile.TemporaryDirectory(prefix='accessible-roadmap-') as d:
    d=pathlib.Path(d); evidence=d/'receipt.json'; pipeline=d/'pipeline.json'
    pipeline.write_text(json.dumps([[sys.executable,'-c','import sys; print("evidence"); sys.exit(7)'],[sys.executable,'-c','import sys; print(sys.stdin.read(),end="")']]))
    preview=json.loads(run([receipt,'--label','offline pipeline','--pipeline',str(pipeline)]));assert preview['state']=='preview'
    assert run([receipt,'--label','offline pipeline','--pipeline',str(pipeline),'--execute','--receipt',str(evidence)],1)=='evidence\n'
    result=json.loads(evidence.read_text());assert [s['exit_code'] for s in result['stages']]==[7,0]
    assert 'Outcome verified: no' in run([rcdo,'receipt-review','--input',str(evidence)],10)
    source=d/'config.json';state=d/'navigation.json';source.write_text('{"service":{"port":80}}')
    run([rcdo,'config-walk','start','--input',str(source),'--state',str(state)])
    run([rcdo,'config-walk','goto','--path','$.service.port','--state',str(state)])
    assert '$.service.port' in run([rcdo,'config-walk','show','--state',str(state)])
    digest=hashlib.sha256(source.read_bytes()).hexdigest()
    source.write_text('{"service":{"port":81}}')
    edit=[rcdo,'config-set','--input',str(source),'--path','service.port','--value','82','--write','--expect-sha256']
    run(edit+[digest],2);assert json.loads(source.read_text())['service']['port']==81
    run(edit+[hashlib.sha256(source.read_bytes()).hexdigest(),'--expect-value','81'])
    assert json.loads(source.read_text())['service']['port']==82
print('PASS: preview, hidden pipeline failure, receipt review, persistent navigation, stale edit refusal, guarded write')
