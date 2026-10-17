#!/usr/bin/env python3
"""真实 API / Worker / Scheduler 报表闭环；仅创建随机测试用户和其资源。"""
import json, os, secrets, time, urllib.request, urllib.error
base = os.environ.get('SMOKE_API_URL', 'http://127.0.0.1:5558')
def request(method, path, body=None, token=None, key=None):
    headers = {'Content-Type': 'application/json'}
    if token: headers['Authorization'] = 'Bearer ' + token
    if key: headers['Idempotency-Key'] = key
    req = urllib.request.Request(base + path, data=None if body is None else json.dumps(body).encode(), headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=5) as r: return r.status, r.read()
    except urllib.error.HTTPError as e: return e.code, e.read()
def ok(method, path, body=None, token=None, key=None, status=200):
    code, raw = request(method, path, body, token, key)
    assert code == status, (path, code, raw)
    return json.loads(raw).get('data', {})
def account():
    username = 'smoke_' + secrets.token_hex(5)
    password = 'Aa9!' + secrets.token_hex(12)
    ok('POST', '/v1/users', {'username':username,'password':password,'nickname':'smoke','email':username+'@example.invalid','phone':'138'+str(secrets.randbelow(100000000)).zfill(8)}, status=200)
    login = ok('POST', '/v1/auth/login', {'username':username,'password':password})
    return login['accessToken']
owner, other = account(), account()
key = secrets.token_hex(12)
task = ok('POST','/v1/reports',{'days':7},owner,key,status=202)
task_id = task['task_id']
assert ok('POST','/v1/reports',{'days':7},owner,key,status=202)['task_id'] == task_id
assert request('POST','/v1/reports',{'days':8},owner,key)[0] == 409
for path in ['/v1/tasks/'+task_id, '/v1/tasks/'+task_id+'/result']:
    assert request('GET',path,token=other)[0] == 403
end = time.monotonic() + 40
while time.monotonic() < end:
    state = ok('GET','/v1/tasks/'+task_id,token=owner)
    if state['status'] == 'succeeded': break
    assert state['status'] not in ('failed','canceled'), state
    time.sleep(.2)
else: raise AssertionError('report execution timed out')
code, csv = request('GET','/v1/tasks/'+task_id+'/result',token=owner)
assert code == 200 and b'user_id' in csv and b'password' not in csv and b'token' not in csv
end = time.monotonic() + 35
while time.monotonic() < end:
    notices = ok('GET','/v1/report-notifications',token=owner)['items']
    if any(x['task_id']==task_id for x in notices): break
    time.sleep(.2)
else: raise AssertionError('Kafka report notification did not arrive')
plan = ok('POST','/v1/schedules',{'name':'smoke periodic','days':1,'cron':'* * * * *','timezone':'UTC','misfire_policy':'skip','concurrency_policy':'forbid'},owner,status=201)
assert request('GET','/v1/schedules/'+plan['schedule_id'],token=other)[0] == 403
end = time.monotonic() + 75
while time.monotonic() < end:
    history = ok('GET','/v1/schedules/'+plan['schedule_id']+'/history',token=owner)['items']
    created = next((x for x in history if x['outcome']=='created'), None)
    if created: break
    time.sleep(.5)
else: raise AssertionError('periodic report did not trigger')
end = time.monotonic() + 40
while time.monotonic() < end:
    periodic = ok('GET','/v1/tasks/'+created['task_id'],token=owner)
    if periodic['status'] == 'succeeded': break
    assert periodic['status'] not in ('failed','canceled'), periodic
    time.sleep(.2)
else: raise AssertionError('periodic report execution timed out')
code, periodic_csv = request('GET','/v1/tasks/'+created['task_id']+'/result',token=owner)
assert code == 200 and b'user_id' in periodic_csv
plan = ok('GET','/v1/schedules/'+plan['schedule_id'],token=owner)
ok('POST','/v1/schedules/'+plan['schedule_id']+'/archive',{'version':plan['version']},owner)
print('PASS report submit/idempotency/ownership/shared result/Kafka notification/periodic scheduler/archive')
