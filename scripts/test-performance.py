#!/usr/bin/env python3
"""Bounded local load/recovery acceptance; accepts no external target URL.

Only synthetic data in this script's disposable Compose project is modified.
Reported times measure this fixture, never an SLA or production RPO/RTO.
"""
import base64,concurrent.futures,csv,gzip,hashlib,http.client,json,os,platform,secrets,subprocess,tempfile,threading,time,uuid,zipfile
from pathlib import Path
from urllib.parse import urlsplit

ROOT=Path(__file__).resolve().parents[1]
OUT=ROOT/'.cache/evidence/performance'
MIB=1024*1024


class Client:
    def __init__(self,port):self.port,self.cookies,self.csrf=port,{},''
    def request(self,method,path,value=None,headers=None,sink=None):
        values={'Origin':'http://localhost:8090','Cookie':'; '.join(k+'='+v for k,v in self.cookies.items()),'X-CSRF-Token':self.csrf,**(headers or {})}
        if isinstance(value,dict):value=json.dumps(value).encode();values['Content-Type']='application/json'
        conn=http.client.HTTPConnection('127.0.0.1',self.port,timeout=180)
        try:
            conn.request(method,urlsplit(path).path,body=value,headers=values)
            response=conn.getresponse();result=dict(response.getheaders())
            for header,value in response.getheaders():
                if header.lower()=='set-cookie':
                    name,content=value.split(';',1)[0].split('=',1);self.cookies[name]=content
            if sink is None:body=response.read(16*MIB)
            else:
                body=b''
                while chunk:=response.read(MIB):sink.write(chunk)
            return response.status,body,result
        finally:conn.close()
    def expect(self,method,path,status,value=None,headers=None,sink=None):
        got,body,heads=self.request(method,path,value,headers,sink)
        if got!=status:raise RuntimeError(f'{method} {path.split("?")[0]} expected {status}, got {got}')
        return json.loads(body) if heads.get('Content-Type','').startswith('application/json') and body else body,heads


class Digest:
    def __init__(self):self.hash=hashlib.sha256();self.size=0
    def write(self,data):self.hash.update(data);self.size+=len(data)


def test_load_and_recovery():
    size_mib=int(os.environ.get('FILEX_PERF_MIB','32'))
    rows=int(os.environ.get('FILEX_PERF_ROWS','2000'))
    if not 8<=size_mib<=8192 or not 100<=rows<=40000:raise ValueError('Fixture outside explicit bounds')
    OUT.mkdir(parents=True,exist_ok=True)
    project='filex-performance-'+uuid.uuid4().hex[:10]
    env=dict(os.environ,POSTGRES_PASSWORD=secrets.token_hex(24),MASTER_KEK=base64.b64encode(secrets.token_bytes(32)).decode())
    measurements=[];stop=threading.Event();samples=[];thread=None
    hardware=json.loads(subprocess.check_output(['docker','info','--format','{{json .}}'],text=True))
    evidence={'profile':{'payloadMiB':size_mib,'exportRows':rows,'chunkBytes':4*MIB,'concurrency':8,'providers':['local','synthetic-s3-tls']},'hardware':{k:hardware.get(k) for k in ['OSType','Architecture','NCPU','MemTotal','KernelVersion','Driver']},'hostPlatform':platform.platform(),'measurements':measurements}
    with tempfile.TemporaryDirectory(prefix=project+'-',dir=ROOT/'.cache') as directory:
        work=Path(directory);certs=work/'certs';certs.mkdir()
        def openssl(*args):subprocess.run(['openssl',*args],cwd=certs,check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        (certs/'ca.cnf').write_text('[req]\ndistinguished_name=dn\nx509_extensions=ca\nprompt=no\n[dn]\nCN=FileX test CA\n[ca]\nbasicConstraints=critical,CA:TRUE\nkeyUsage=critical,keyCertSign,cRLSign\n')
        openssl('req','-x509','-newkey','rsa:2048','-nodes','-keyout','ca.key','-out','ca.crt','-days','1','-config','ca.cnf')
        openssl('req','-newkey','rsa:2048','-nodes','-keyout','server.key','-out','server.csr','-subj','/CN=s3-fixture')
        (certs/'extensions.cnf').write_text('subjectAltName=DNS:s3-fixture\nextendedKeyUsage=serverAuth\nbasicConstraints=critical,CA:FALSE\n')
        openssl('x509','-req','-in','server.csr','-CA','ca.crt','-CAkey','ca.key','-CAcreateserial','-out','server.crt','-days','1','-extfile','extensions.cnf')
        (certs/'server.key').chmod(0o644)
        config=json.loads(subprocess.check_output(['docker','compose','--env-file','/dev/null','--project-name',project,'-f',str(ROOT/'deploy/compose.yaml'),'config','--format','json'],env=env,text=True))
        config['services']['traefik']['ports']=['127.0.0.1::8080']
        for name in ['api','worker']:
            config['services'][name]['environment']['SSL_CERT_FILE']='/fixtures/ca.crt'
            config['services'][name]['volumes'].append({'type':'bind','source':str(certs),'target':'/fixtures','read_only':True})
        config['services']['s3-fixture']={'build':{'context':str(ROOT),'dockerfile':'scripts/fixtures/Dockerfile.s3'},'networks':['backend'],'volumes':['s3_data:/data',str(certs)+':/fixtures:ro'],'read_only':True,'cap_drop':['ALL'],'security_opt':['no-new-privileges:true'],'mem_limit':'256m'}
        config['volumes']['s3_data']={}
        profile=work/'compose.json';profile.write_text(json.dumps(config));profile.chmod(0o600)
        def dc(*args,**kwargs):return subprocess.run(['docker','compose','--project-name',project,'-f',str(profile),*args],env=env,cwd=ROOT,check=True,text=True,**kwargs)
        def cli(*args,value=None):return json.loads(dc('exec','-T','api','filex-cli',*args,'--json',input=json.dumps(value) if value is not None else None,capture_output=True).stdout)
        def sql(statement):return dc('exec','-T','postgres','psql','-v','ON_ERROR_STOP=1','-U','filex','-d','filex','-At','-c',statement,capture_output=True).stdout.strip()
        def until(fn,timeout=180):
            end=time.monotonic()+timeout
            while time.monotonic()<end:
                try:
                    if fn():return
                except (OSError,http.client.HTTPException):pass
                time.sleep(.2)
            raise RuntimeError('Fixture condition timed out')
        def record(name,started,**fields):
            elapsed=time.monotonic()-started
            item={'name':name,'seconds':round(elapsed,3),**fields}
            if 'bytes' in fields:item['MiBPerSecond']=round(fields['bytes']/MIB/elapsed,3)
            measurements.append(item);print(json.dumps(item),flush=True)
        try:
            dc('up','--detach','--build','--wait','--wait-timeout','120')
            port=int(dc('port','traefik','8080',capture_output=True).stdout.strip().rsplit(':',1)[1]);owner=Client(port)
            def ready():
                try:return owner.request('GET','/api/v1/status')[0]==200
                except OSError:return False
            until(ready)
            containers=dc('ps','--quiet',capture_output=True).stdout.split()
            def sample():
                while not stop.is_set():
                    result=subprocess.run(['docker','stats','--no-stream','--format','{{json .}}',*containers],capture_output=True,text=True)
                    if result.returncode==0:samples.extend(dict(json.loads(line),sampleAtUnix=time.time()) for line in result.stdout.splitlines())
                    stop.wait(.5)
            thread=threading.Thread(target=sample,daemon=True);thread.start()
            password=secrets.token_urlsafe(24)
            cli('user','create',value={'email':'load@example.test','password':password,'role':'admin'})
            login,_=owner.expect('POST','/api/v1/auth/login',200,{'email':'load@example.test','password':password});owner.csrf=login['csrfToken']
            principal,_=owner.expect('GET','/api/v1/auth/me',200);uid=principal['id']
            transfer,_=owner.expect('POST','/api/v1/transfers',201);tid=transfer['id']
            block=secrets.token_bytes(4*MIB);digest=Digest();started=time.monotonic()
            _,headers=owner.expect('POST',f'/api/v1/transfers/{tid}/uploads',201,headers={'Tus-Resumable':'1.0.0','Upload-Length':str(size_mib*MIB),'Upload-Metadata':'filename '+base64.b64encode(b'large.bin').decode()})
            upload=headers['Location']
            for offset in range(0,size_mib*MIB,len(block)):
                chunk=block[:min(len(block),size_mib*MIB-offset)]
                owner.expect('PATCH',upload,204,chunk,{'Tus-Resumable':'1.0.0','Upload-Offset':str(offset),'Content-Type':'application/offset+octet-stream'});digest.write(chunk)
                if offset==0:
                    _,heads=owner.expect('HEAD',upload,200,headers={'Tus-Resumable':'1.0.0'})
                    if int(heads['Upload-Offset'])!=len(chunk):raise RuntimeError('Resume offset differs')
            owner.expect('POST',upload+'/finalize',204);record('upload-finalize',started,bytes=digest.size,sha256=digest.hash.hexdigest())
            data,_=owner.expect('GET',f'/api/v1/transfers/{tid}',200)
            source=sql(f"SELECT provider_id FROM files WHERE transfer_id='{tid}'")
            owner.expect('POST','/api/v1/auth/reauth',204,{'password':password})
            target,_=owner.expect('POST','/api/v1/admin/storage',201,{'kind':'s3','purpose':'payload','priority':10,'s3':{'endpoint':'https://s3-fixture:8443','region':'fixture','bucket':'fixture','accessKey':'fixture-access','secretKey':secrets.token_hex(24)}})
            started=time.monotonic()
            job,_=owner.expect('POST','/api/v1/admin/storage/migrations',201,{'sourceId':source,'targetId':target['id'],'dryRun':False})
            owner.expect('PATCH','/api/v1/admin/storage/migrations/'+job['id'],204,{'state':'running'})
            until(lambda:sql(f"SELECT state FROM storage_migrations WHERE id='{job['id']}'")=='completed',600);record('local-to-s3-tls',started)
            owner.expect('POST','/api/v1/auth/reauth',204,{'password':password})
            started=time.monotonic();key,_=owner.expect('POST','/api/v1/admin/keys/rotate',202,{'class':'payload'})
            until(lambda:sql(f"SELECT count(*) FROM files WHERE transfer_id='{tid}' AND key_id='{key['keyId']}'")=='1',600);record('payload-key-rotation',started)
            published,_=owner.expect('POST',f'/api/v1/transfers/{tid}/publish',200,{'expectedRevision':data['revision'],'ttlSeconds':3600,'commonLink':True})
            token=published['links'][0]['url'].split('#token=')[1]
            recipient=Client(port);recipient.expect('POST','/api/v1/recipient/session',200,{'token':token})
            refs,_=recipient.expect('GET','/api/v1/recipient/files',200)
            dc('stop','redis')
            with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
                claims=list(pool.map(lambda _:recipient.request('POST','/api/v1/recipient/downloads',{'fileRef':refs[0]['ref']}),range(8)))
            if sum(c[0]==201 for c in claims)!=1 or any(c[0] not in [201,409] for c in claims):raise RuntimeError('Redis loss/concurrent acquire broke single winner')
            lease=json.loads(next(c[1] for c in claims if c[0]==201))['url']
            part=Digest();recipient.expect('GET',lease,206,headers={'Range':'bytes=0-1048575'},sink=part)
            if part.hash.digest()!=hashlib.sha256(block[:MIB]).digest():raise RuntimeError('Range differs')
            dc('start','redis')
            # Synthetic volume deliberately exceeds tiny smoke fixtures; no live data is imported.
            sql(f"INSERT INTO transfers(id,owner_id,title) SELECT gen_random_uuid(),'{uid}','Load row '||n FROM generate_series(1,{rows}) n")
            # Block the snapshot write in this fixture, then kill the real worker
            # after its claim is committed. This makes the fault deterministic.
            blocker=subprocess.Popen(['docker','compose','--project-name',project,'-f',str(profile),'exec','-T','-e','PGAPPNAME=filex-load-blocker','postgres','psql','-v','ON_ERROR_STOP=1','-U','filex','-d','filex'],env=env,stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,text=True)
            try:
                blocker.stdin.write("BEGIN;\nLOCK TABLE export_rows IN ACCESS EXCLUSIVE MODE;\nSELECT pg_sleep(120);\n");blocker.stdin.flush()
                until(lambda:sql("SELECT count(*) FROM pg_locks WHERE relation='export_rows'::regclass AND mode='AccessExclusiveLock' AND granted")=='1')
                started=time.monotonic();job,_=owner.expect('POST','/api/v1/exports',202,{'kind':'user','format':'csv','compression':'gzip'})
                until(lambda:sql(f"SELECT state FROM export_jobs WHERE id='{job['id']}'")=='snapshotting')
                dc('kill','--signal','SIGKILL','worker')
            finally:
                sql("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE application_name='filex-load-blocker'")
                blocker.stdin.close();blocker.wait(timeout=10)
            # Advance only the synthetic expired lease; this is fault injection,
            # not a measurement of the real three-minute crash-recovery delay.
            sql(f"UPDATE export_jobs SET lease_until=clock_timestamp()-interval '1 second' WHERE id='{job['id']}'")
            dc('start','worker')
            until(lambda:sql(f"SELECT state FROM export_jobs WHERE id='{job['id']}'")=='completed',180)
            artifact=sql(f"SELECT artifact_id FROM export_jobs WHERE id='{job['id']}'")
            credential,_=owner.expect('POST',f'/api/v1/artifacts/{artifact}/credentials',201)
            claimed,_=owner.expect('POST',f'/api/v1/artifacts/{artifact}/claim',200,credential)
            export_path=work/'synthetic-export.csv.gz'
            with export_path.open('wb') as output:owner.expect('GET',claimed['url'],200,sink=output)
            with gzip.open(export_path,'rt',newline='') as data:
                actual_rows=sum(1 for _ in csv.DictReader(data))
            if actual_rows!=rows+1:raise RuntimeError('Recovered export lost or duplicated rows')
            sql('VACUUM (ANALYZE) transfers')
            retained=sql("SELECT count(*) FROM pg_stat_activity WHERE state='idle in transaction' AND backend_xmin IS NOT NULL")
            if retained!='0':raise RuntimeError('Export retained a database snapshot')
            record('export-crash-retry-and-vacuum',started,rows=actual_rows,idleSnapshots=int(retained),faultInjectedLeaseExpiry=True)
            latencies=[]
            def probe(_):
                start=time.monotonic();owner.expect('GET','/api/v1/status',200);return (time.monotonic()-start)*1000
            with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:latencies=list(pool.map(probe,range(100)))
            latencies.sort();evidence['statusLatencyMs']={'p50':latencies[49],'p95':latencies[94],'max':latencies[-1],'requests':100}
            dc('stop','worker')
            provider=sql("SELECT id FROM storage_providers WHERE purpose='backups' AND enabled LIMIT 1")
            started=time.monotonic();backup=cli('backup','create',value={'providerId':provider,'reason':'Local load and recovery acceptance'})
            cli('backup','verify',value={'backupId':backup['id']});record('backup-verify',started)
            late,_=owner.expect('POST','/api/v1/transfers',201)
            started=time.monotonic();cli('backup','restore',value={'providerId':provider,'backupId':backup['id'],'reason':'Local recovery with multi-chunk payload and metadata'})
            owner.expect('GET','/api/v1/auth/me',503)
            dc('restart','api','worker');until(lambda:owner.request('GET','/api/v1/auth/me')[0]==401)
            login,_=owner.expect('POST','/api/v1/auth/login',200,{'email':'load@example.test','password':password});owner.csrf=login['csrfToken']
            owner.expect('GET',f'/api/v1/transfers/{tid}',503)
            reconciliation=cli('recovery','reconcile',value={'reason':'Verify audit crypto storage and restored capabilities'})
            state=cli('operations','status');job=next(j for j in state['jobs'] if j['id']==reconciliation['id'])
            report=cli('recovery','report',value={'reconciliationId':job['result']['reconciliationId']})
            if not report['passed'] or report['findings']:raise RuntimeError('Recovery findings block acceptance')
            cli('recovery','reopen',value={'reconciliationId':report['id'],'hash':report['hash'],'reason':'Reviewed disposable restored capabilities','acknowledgeCapabilities':True})
            if sql(f"SELECT count(*) FROM transfers WHERE id='{late['id']}'")!='0':raise RuntimeError('Restore did not roll back fixture metadata')
            record('restore-restart-reconcile-reopen',started,files=report['files'],objects=report['objects'])
            # Restore invalidates recipient sessions/leases. Explicitly authenticate/acquire again.
            recipient=Client(port);recipient.expect('POST','/api/v1/recipient/session',200,{'token':token})
            lease,_=recipient.expect('POST','/api/v1/recipient/downloads',201,{'all':True})
            started=time.monotonic();archive=work/'synthetic.zip'
            with archive.open('wb') as output:recipient.expect('GET',lease['url'],200,sink=output)
            with zipfile.ZipFile(archive) as zipped:
                received=Digest()
                with zipped.open('large.bin') as data:
                    while chunk:=data.read(MIB):received.write(chunk)
                if received.size!=digest.size or received.hash.digest()!=digest.hash.digest():raise RuntimeError('Restored/rotated ZIP payload differs')
            record('download-zip-verify',started,bytes=received.size)
            recipient.expect('GET',lease['url'],401)
            cli('audit','verify')
            inspected=json.loads(subprocess.check_output(['docker','inspect',*containers],text=True))
            if any(c['State'].get('OOMKilled') for c in inspected):raise RuntimeError('Container exceeded memory bound')
            evidence['assertions']={'resumableUpload':True,'oneWinnerWithoutRedis':True,'localToS3TLS':True,'rotationPreservedSHA256':True,'workerSIGKILLRecovery':True,'exportRowsVerified':True,'exportReleasedSnapshot':True,'verifiedBackup':True,'oldSessionsRevoked':True,'closedUntilReviewed':True,'restoredMetadata':True,'zipSHA256':True,'oneTimeCompletion':True,'auditVerified':True,'noOOM':True}
        finally:
            stop.set()
            if thread:thread.join(timeout=15)
            evidence['containerSamples']=samples
            (OUT/'results.json').write_text(json.dumps(evidence,indent=2)+'\n')
            (OUT/f'results-{size_mib}mib-{rows}rows.json').write_text(json.dumps(evidence,indent=2)+'\n')
            dc('down','--volumes','--remove-orphans')
    print('PASS: bounded local load/recovery acceptance',flush=True)


if __name__=='__main__':test_load_and_recovery()
