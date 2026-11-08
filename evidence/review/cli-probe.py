import json, subprocess, sys, selectors
binary=sys.argv[1]
cases=[
 ('truncated_object',['--stream','a'],'{"a":1}{"a":'),
 ('truncated_array',['--stream','a'],'{"a":1}[1,'),
 ('invalid_closing_token',['--stream','a'],'{"a":1}]'),
 ('real_serialization_error',['--stream','--compact','[a, 1.0 / a]'],'{"a":1}{"a":0}{"a":2}'),
 ('corrected_input',['--stream','a'],'{"a":2}'),
 ('ordinary_yaml',['-i','json','-o','yaml'],'{"a":1}'),
]
for name,args,inp in cases:
 p=subprocess.run([binary,'query',*args],input=inp,text=True,capture_output=True,timeout=5)
 print(json.dumps(dict(case=name,input=inp,args=args,exitCode=p.returncode,stdout=p.stdout,stderr=p.stderr),ensure_ascii=False))
p=subprocess.Popen([binary,'query','--stream','name'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
try:
 p.stdin.write(b'{"name":"live"}');p.stdin.flush()
 sel=selectors.DefaultSelector();sel.register(p.stdout,selectors.EVENT_READ)
 ready=bool(sel.select(3));line=p.stdout.readline() if ready else b''
 print(json.dumps(dict(case='real_binary_open_pipe',inputStillOpen=True,outputReady=ready,stdout=line.decode())))
 p.stdin.close();code=p.wait(timeout=5)
 assert ready and line==b'"live"\n' and code==0
finally:
 if p.poll() is None:p.kill();p.wait()
