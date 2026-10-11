#!/usr/bin/env python3
"""Build the reviewed API contract. --check detects drift; route coverage is tested separately."""
import copy,json,re,sys
from pathlib import Path
root=Path(__file__).resolve().parents[1]
S={'type':'string'};I={'type':'integer','minimum':0};B={'type':'boolean'};U={'type':'string','format':'uuid'};T={'type':'string','format':'date-time'};H={'type':'string','pattern':'^[0-9a-f]{64}$'}
def ref(name):return {'$ref':'#/components/schemas/'+name}
def enum(*values):return {'type':'string','enum':list(values)}
def null(schema):return {'anyOf':[schema,{'type':'null'}]}
def arr(schema):return {'type':'array','items':schema}
def obj(props,required=None):return {'type':'object','additionalProperties':False,'properties':props,'required':list(props) if required is None else required}
D={}
def define(name,props,required=None):D[name]=obj(props,required);return ref(name)
Error=define('Error',{'code':enum('unauthenticated','forbidden','not_found','conflict','invalid_input','rate_limited','internal_error'),'message':S})
ID=define('Identifier',{'id':U});Draft=define('Draft',{'id':U,'revision':I});Status=define('Status',{'status':enum('operational','unavailable')})
Principal=define('Principal',{'id':U,'email':{'type':'string','format':'email'},'role':enum('user','admin'),'locale':enum('ru','en'),'sessionId':U},['id','email','role','locale'])
Session=define('Session',{'csrfToken':S,'user':Principal,'expiresAt':T})
Settings=define('Settings',{'version':{'const':1},'publicOrigin':S,'sessionTtlSeconds':{'type':'integer','minimum':300,'maximum':604800},'transferMaxTtlSeconds':{'type':'integer','minimum':1,'maximum':31536000},'maxFiles':{'type':'integer','minimum':1,'maximum':100000},'maxPathBytes':{'type':'integer','minimum':1,'maximum':4096},'fileTypeMode':enum('disabled','allow','deny'),'fileTypes':arr(S),'bannerRu':{'type':'string','maxLength':2000},'bannerEn':{'type':'string','maxLength':2000}},['version','transferMaxTtlSeconds','maxFiles','maxPathBytes'])
FileState=enum('uploading','finalizing','ready','leased','cancelled','rejected','integrity_failed','storage_missing','deleting','deleted')
TransferState=enum('draft','active','completed','expired','revoked')
File=define('File',{'id':U,'transferId':S,'path':S,'size':I,'offset':I,'state':FileState,'sha256':null(H)})
Link=define('Link',{'id':U,'email':null(S),'url':S})
Transfer=define('Transfer',{'id':U,'revision':I,'title':S,'state':TransferState,'manifestHash':null(H),'files':arr(File),'links':arr(Link)})
Published=define('Published',{'id':U,'revision':I,'publishedAt':T,'expiresAt':T,'absoluteExpiresAt':T,'manifestHash':H})
Filter=define('Filter',{'query':{'type':'string','maxLength':500},'state':{'type':'string','enum':['']+TransferState['enum']},'after':S,'viewId':S},[])
Search=define('SearchResult',{'items':arr(obj({'id':U,'title':S,'state':TransferState,'revision':I,'createdAt':T})),'nextCursor':S})
Journal=define('JournalEntry',{'occurredAt':T,'fileCount':I,'totalBytes':I,'status':S})
Preference=define('NotificationPreferences',{'locale':enum('ru','en'),'timezone':S,'digest':enum('immediate','hourly','daily','weekly','off'),'quietStart':null({'type':'integer','minimum':0,'maximum':23}),'quietEnd':null({'type':'integer','minimum':0,'maximum':23}),'browserEnabled':B})
Event=define('OwnerEvent',{'schemaVersion':{'const':1},'id':I,'kind':enum('transfer.published','transfer.expiring','transfer.expired','transfer.revoked','download.completed'),'transferId':null(U),'createdAt':T,'readAt':null(T)})
Webhook=define('WebhookEvent',{'schemaVersion':{'const':1},'eventId':U,'type':D['OwnerEvent']['properties']['kind'],'transferId':U})
Delivery=define('Delivery',{'id':U,'eventType':S,'kind':enum('smtp','webhook'),'state':enum('pending','sending','delivered','dead','cancelled'),'attempts':I,'lastCode':null(S),'createdAt':T})
ReportKind=enum('user','admin');ReportState=enum('active','trashed','purged');RenderState=enum('pending','running','ready','failed')
Report=define('Report',{'id':U,'transferId':U,'kind':ReportKind,'revision':I,'canonicalHash':H,'state':ReportState,'renderState':RenderState,'createdAt':T})
Snapshot=define('ReportSnapshot',{'schemaVersion':{'const':1},'reportId':U,'documentContentHash':H,'revision':I,'kind':ReportKind,'transferId':U,'title':S,'state':TransferState,'publishedAt':null(T),'manifestHash':null(H),'createdAt':T,'locale':enum('ru','en'),'timezone':S,'manifest':{'$ref':'schemas/manifest-v1.json'},'files':arr(obj({'path':S,'size':I,'sha256':null(H),'state':FileState})),'downloads':arr(Journal),'administrativeEvents':arr(obj({'downloadId':U,'recipientEmail':null(S),'kind':enum('file','zip'),'state':S,'createdAt':T,'completedAt':null(T)}))},['schemaVersion','reportId','revision','kind','transferId','title','state','publishedAt','manifestHash','createdAt','locale','timezone','files','downloads'])
ReportDetail=define('ReportDetail',{'id':U,'state':ReportState,'renderState':RenderState,'snapshot':Snapshot,'canonicalHash':H,'artifactHashes':{'type':'object','additionalProperties':H},'artifacts':{'type':'object','additionalProperties':U},'history':arr(obj({'action':enum('created','active','trashed','purged'),'createdAt':T}))},['id','state','renderState','artifacts','history'])
ExportState=enum('queued','snapshotting','materialized','rendering','completed','failed','cancelled')
Export=define('Export',{'id':U,'kind':ReportKind,'format':enum('json','csv'),'compression':enum('auto','none','gzip','zip'),'state':ExportState,'rowCount':I,'processed':I,'snapshotAt':null(T),'artifactId':null(U),'errorCode':null(S),'createdAt':T})
Bulk=define('Bulk',{'id':U,'action':enum('revoke','cancel_upload'),'state':enum('paused','queued','running','completed','cancelled'),'total':I,'done':I,'conflicts':I})
SMTP=define('SMTPConfiguration',{'host':S,'port':{'type':'integer','minimum':1,'maximum':65535},'tls':enum('starttls','smtps'),'username':S,'password':S,'from':{'type':'string','format':'email'}},['host','port','tls','from'])
S3=define('S3Configuration',{'endpoint':S,'region':S,'bucket':S,'accessKey':S,'secretKey':S,'sessionToken':S},['region','bucket','accessKey','secretKey'])
Storage=define('StorageProvider',{'id':U,'kind':enum('local','s3'),'purpose':enum('payload','artifacts','backups'),'priority':{'type':'integer'},'enabled':B,'draining':B})
Reason={'type':'string','minLength':5,'maxLength':2000}
password={'type':'string','maxLength':1024,'description':'Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits.'}
ops={}
def add(method,path,title,response=None,body=None,code=None,description=''):
 ops[(method,'/api/v1'+path)]={'title':title,'response':response,'body':body,'code':code or (204 if response is None else 200),'description':description}
add('GET','/status','Read aggregate service status',Status,description='Current database reachability only; no topology, history or component identifiers. Does not prove production readiness.')
add('GET','/banner','Read the global announcement',obj({'ru':S,'en':S}))
add('POST','/auth/login','Sign in with email and password',Session,obj({'email':S,'password':password}),description='Requires exact publicOrigin. Sets HttpOnly SameSite=Strict filexSession cookie. Generic errors and durable per-IP/per-email throttling prevent enumeration. Store csrfToken for unsafe requests.')
add('GET','/auth/me','Read current principal',Principal)
for path,title,body in [('/auth/logout','End the current session',None),('/auth/reauth','Reauthenticate for five minutes',obj({'password':password})),('/auth/password','Change password and invalidate sessions and tokens',obj({'currentPassword':password,'newPassword':password}))]:add('PATCH' if path.endswith('password') else 'POST',path,title,body=body)
add('GET','/auth/sessions','List own sessions',arr(obj({'id':U,'createdAt':T,'expiresAt':T,'revokedAt':null(T)})))
add('DELETE','/auth/sessions/{id}','Revoke an owned session')
add('GET','/auth/history','List own login history',arr(obj({'occurredAt':T,'success':B})))
add('GET','/auth/tokens','List own API token metadata',arr(obj({'id':U,'scopes':arr(S),'expiresAt':T,'revokedAt':null(T)})))
add('POST','/auth/tokens','Create a scoped API token',obj({'id':U,'token':S}),obj({'scopes':arr(S),'ttlSeconds':{'type':'integer','minimum':60,'maximum':2592000}}),201,'Cookie session and fresh reauthentication required; token is returned once. No privileged ADMIN scopes may be minted.')
add('DELETE','/auth/tokens/{id}','Revoke an owned API token')
add('POST','/admin/users','Provision an account',ID,obj({'email':S,'password':password,'role':enum('user','admin')}),201)
add('GET','/admin/settings','List configuration revisions',arr(obj({'revision':I,'active':B,'validated':B,'value':Settings})))
add('POST','/admin/settings','Create a typed configuration draft',obj({'revision':I}),Settings,201)
add('POST','/admin/settings/{revision}/validate','Validate a configuration revision')
add('POST','/admin/settings/{revision}/apply','Apply or roll back to a validated revision',body=obj({'expectedRevision':I}),description='Compare-and-set active revision, with an audit record in the same transaction. Select a previous validated revision to roll back.')
add('POST','/admin/audit/verify','Verify the full HMAC audit chain',obj({'status':{'const':'verified'}}))
add('POST','/transfers','Create a draft',Draft,code=201)
add('GET','/transfers','List own transfers by ID cursor',obj({'items':arr(obj({'id':U,'state':TransferState,'revision':I,'title':S,'manifestHash':null(H)})),'nextCursor':S}))
add('POST','/transfers/search','Search own transfers using a private request body',Search,Filter,description='Literal case-insensitive title search, state filter and stable createdAt/id cursor; no raw search text in the URL. A saved view is owner-scoped.')
add('GET','/transfers/{id}','Read an owned transfer with links',Transfer)
add('PATCH','/transfers/{id}','Edit permitted transfer metadata',body=obj({'title':{'type':'string','maxLength':300},'message':{'type':'string','maxLength':5000}},[]),description='Full metadata replacement; omitted title/message becomes empty. Payload names and paths remain immutable after publication.')
add('POST','/transfers/{id}/publish','Publish an immutable manifest and recipient links',obj({'transfer':Published,'links':arr(Link)}),obj({'expectedRevision':I,'ttlSeconds':{'type':'integer','minimum':1},'recipients':{'type':'array','maxItems':100,'items':obj({'email':S,'password':password},['email'])},'commonLink':B,'commonPassword':password},['expectedRevision','ttlSeconds']),description='Requires at least one ready file, no unfinished files and either a common link or recipients. Exact-input retry is idempotent. Changed revision/TTL/recipients/passwords conflicts. TTL starts at publication. Commit includes links, manifest, audit and outbox; it does not imply email delivery.')
add('POST','/transfers/{id}/revoke','Revoke recipient access and schedule payload deletion',description='Epoch changes interrupt active streams. Holds continue to prevent physical deletion.')
add('OPTIONS','/transfers/{id}/uploads','Discover tus capabilities')
add('POST','/transfers/{id}/uploads','Create a sticky resumable tus upload',Draft,code=201,description='tus 1.0 creation/termination, maximum file size 1 TiB. Upload-Metadata filename is a Base64 UTF-8 relative path. The selected provider and key remain sticky. Location identifies the resource.')
add('OPTIONS','/uploads/{id}','Discover tus resource capabilities')
add('HEAD','/uploads/{id}','Read the authoritative upload offset',description='Returns Upload-Offset and Upload-Length. Does not write bytes or finalize.')
add('PATCH','/uploads/{id}','Append up to 4 MiB at the expected offset',description='Content-Type application/offset+octet-stream. Offset conflict returns 409. An empty PATCH is a no-op. Lost response: read HEAD before retrying. POST with X-HTTP-Method-Override: PATCH is equivalent.')
add('DELETE','/uploads/{id}','Cancel a draft upload and queue deletion',description='Also accepts POST with X-HTTP-Method-Override: DELETE. No cancellation after publication.')
add('POST','/uploads/{id}/finalize','Verify server checksum and file policy',description='The server reads authenticated chunks, computes SHA-256/CRC32, sniffs MIME and invokes an optional scanner. Client hashes or ready flags are never accepted.')
add('PATCH','/files/{id}','Rename or move one draft file',body=obj({'path':S}))
add('PATCH','/transfers/{id}/files','Atomically rename or move a selection',body=obj({'moves':{'type':'array','minItems':1,'maxItems':1000,'items':obj({'id':U,'path':S})}}),description='Validates the complete resulting tree; a collision or invalid path rolls back every move. Published payload is immutable.')
add('POST','/transfers/{id}/folders','Create a virtual draft folder',obj({'id':U,'path':S}),obj({'path':S}),201)
add('GET','/transfers/{id}/folders','List virtual folders',arr(obj({'id':U,'path':S})))
add('PATCH','/transfers/{id}/folders/{folder}','Recursively move a draft folder',body=obj({'path':S}))
add('DELETE','/transfers/{id}/folders/{folder}','Delete a draft folder and cancel its uploads',description='All child paths are selected inside the parent transaction. Holds prevent physical cleanup.')
add('POST','/transfers/{id}/links/{link}/rotate','Rotate a recipient capability and password',obj({'url':S}),obj({'password':password},[]),description='Immediately invalidates previous link epochs and recipient sessions. Omitted password removes the password requirement.')
add('DELETE','/transfers/{id}/links/{link}','Revoke one recipient link')
add('GET','/transfers/{id}/links/{link}/qr','Download an owned capability QR code','binary')
add('POST','/transfers/{id}/qr-bundle','Create encrypted ZIP of recipient QR codes and email mapping',obj({'artifactId':U}),code=201,description='Sensitive temporary artifact, acquired through the artifact credential protocol. A concurrent link change rejects creation. Link rotation/revocation and transfer closure invalidate existing QR artifacts immediately.')
add('GET','/transfers/{id}/journal','Read recipient-independent download totals',arr(Journal),description='Contains no recipient identity, link ID, IP or user agent. One completed ZIP produces one aggregate row; totalBytes counts payload bytes.')
add('POST','/recipient/session','Exchange a capability and optional password for a session',obj({'status':{'const':'ready'}}),obj({'token':S,'password':password},['token']),description='Capability remains in URL fragment until the browser submits this request. Sets filexRecipient; exact Origin and per-capability throttling required. Does not consume files.')
add('GET','/recipient/files','List session-bound opaque file references',arr(obj({'ref':S,'path':S,'size':I,'state':FileState})),description='References are stable within a recipient session and unusable in another session. No download acquisition on GET.')
add('POST','/recipient/downloads','Reserve an individual file or atomic ZIP',obj({'id':U,'url':S}),obj({'fileRef':S,'all':B},[]),201,'Choose exactly one of fileRef or all=true. All reserves every remaining ready file atomically; any held/unavailable file blocks the whole ZIP. Leases last five minutes and renew while streaming.')
for method in ['GET','HEAD']:add(method,'/recipient/downloads/{id}/content','Read reserved bytes with Range and fencing' if method=='GET' else 'Inspect reserved content without consumption','binary',description='Single byte range only. Complete successful server-observed range coverage commits logical deletion of the entire reservation. Revoke/expiry/hold guards run before and after each bounded 32 KiB write; already transmitted bytes cannot be retracted. HEAD never consumes.')
add('POST','/admin/transfers/{id}/hold','Apply or release an indefinite retention hold',body=obj({'frozen':B,'fileId':S,'reason':Reason},['frozen','reason']),description='Empty fileId addresses the transfer. Transfer hold pauses ordinary TTL; absolute expiry never extends. USER receives no hold event. Unfreeze never resurrects revoked/expired rights.')
add('POST','/admin/transfers/{id}/forensic-zip','Download a non-consuming forensic ZIP','binary',obj({'reason':Reason}),description='Requires a transfer-wide hold, finalized retained files, current ADMIN session and fresh reauthentication. Loss of hold or privilege interrupts the stream. Writes an administrative audit record.')
add('POST','/admin/files/{id}/forensic','Download forensic plaintext or one raw ciphertext object','binary',obj({'reason':Reason,'rawCiphertext':B,'objectKey':S},['reason']),description='Raw ciphertext requires integrity_failed and a belonging objectKey. Plaintext authenticates every chunk. Neither consumes recipient rights; reason and actor are audited.')
add('POST','/admin/files/{id}/check','Recheck payload integrity and conditionally repair state',obj({'status':S}),obj({'reason':Reason}),description='Restores ready only when bytes match the original checksum and the parent is still active. Never reopens an expired/revoked transfer.')
add('POST','/admin/files/{id}/force-delete','Force-delete retained file content',body=obj({'reason':Reason}),description='Irreversible. Requires transfer hold to be released; explicitly releases the selected file hold. Audited reason and reauthentication required.')
add('POST','/admin/storage','Probe and register a storage provider',ID,obj({'kind':enum('local','s3'),'purpose':enum('payload','artifacts'),'root':S,'s3':S3,'priority':{'type':'integer'}},['kind']),201,'Writes, reads and deletes a generated probe. Configuration secrets are encrypted with the configuration DEK. Only one enabled artifact provider is allowed.')
add('GET','/admin/storage','List redacted provider metadata',arr(Storage))
add('PATCH','/admin/storage/{id}','Change provider priority, drain or enablement',body=obj({'priority':{'type':'integer'},'draining':B,'enabled':B}),description='Drain excludes new uploads; existing sticky uploads and reads continue.')
add('POST','/admin/storage/migrations','Preview or prepare a storage migration',obj({'id':U,'state':{'const':'paused'}}),obj({'sourceId':U,'targetId':U,'dryRun':B}),201,'dryRun=true returns 200 with files and bytes without mutation. Otherwise creates paused migration and drains the source. Target must be enabled, non-draining payload storage.')
add('PATCH','/admin/storage/migrations/{id}','Pause, resume or cancel a storage migration',body=obj({'state':enum('paused','running','cancelled')}))
add('POST','/admin/keys/rotate','Rotate a class key and schedule re-encryption',obj({'id':U,'keyId':U,'state':enum('running','completed')}),obj({'class':enum('payload','artifacts','configuration','signing','backups')}),202,'New writes use the new key immediately. One running/paused rotation per class. Existing keys are retained; signing rotation keeps historical public keys permanently.')
add('PATCH','/admin/keys/rotations/{id}','Pause, resume or cancel background re-encryption',body=obj({'state':enum('paused','running','cancelled')}),description='Does not undo the new active key or delete historical keys. A paused job prevents another rotation of that class.')
add('GET','/admin/maintenance','List migrations and rotations',arr(obj({'id':U,'kind':S,'state':S,'progress':I})))
add('GET','/signing-keys/{id}','Read a historical signing public key',obj({'id':U,'publicKey':S,'algorithm':{'const':'Ed25519'}}),description='publicKey is Base64 raw Ed25519 bytes. Verify trust out of band; the ZIP embeds SPKI PEM.')
add('PUT','/admin/smtp','Save encrypted SMTP transport configuration',obj({'id':U,'status':{'const':'configured'}}),SMTP,description='Only STARTTLS or SMTPS with certificate validation; authentication occurs after TLS. Saving is not proof of delivery.')
add('POST','/webhooks','Create an outbound signed webhook',obj({'id':U,'secret':S}),obj({'url':S,'secret':S},['url']),201,'HTTPS port 443 only, no redirects or proxy. DNS resolved for every connection and all answers must be public IPv4. Empty secret generates a secret returned once. HMAC timestamp + body; consumers must deduplicate the signed eventId.')
add('DELETE','/webhooks/{id}','Disable an owned webhook')
add('GET','/notification-preferences','Read notification preferences',Preference)
add('PUT','/notification-preferences','Set locale, timezone, digest and quiet hours',body=Preference)
add('GET','/notifications','Poll owner-safe notifications',arr(Event))
add('POST','/notifications/{id}/read','Mark an owned notification read')
add('GET','/events','Subscribe to owner-safe WebSocket notifications',code=101,description='Cookie session and exact Origin only. after cursor resumes delivery; each event has schemaVersion=1. Session is rechecked every second. Poll /notifications as fallback. See AsyncAPI.')
add('GET','/deliveries','List own delivery states',arr(Delivery))
add('POST','/deliveries/{id}/retry','Resend a dead or previously delivered message',description='Resets bounded retry attempts; stable event ID allows receiver deduplication.')
add('POST','/transfers/{id}/reports','Create an immutable report revision',obj({'id':U,'canonicalHash':H,'renderState':{'const':'pending'}}),obj({'kind':ReportKind,'locale':enum('ru','en'),'timezone':S}),201,'Snapshot uses REPEATABLE READ. USER snapshot excludes recipient correlation. ADMIN requires recent reauthentication and may include recipient association. PDF/CSV/JSON render asynchronously from the immutable encrypted snapshot.')
add('GET','/reports','List owned report revisions',arr(Report))
add('GET','/reports/{id}','Read a report snapshot, artifact IDs and history',ReportDetail)
add('POST','/reports/{id}/state','Trash, restore or purge a report',body=obj({'state':ReportState}),description='Purge requires trashed state, deletes encrypted content asynchronously and preserves canonicalHash/history tombstones. Purged reports cannot be restored.')
add('POST','/reports/{id}/retry','Retry rendering or regenerate expired report artifacts',description='Uses the existing immutable snapshot, never a new live-data snapshot. Available for failed rendering or expired/missing ready artifacts.')
add('GET','/verify/reports/{id}','Read the hash of an active USER report',obj({'canonicalHash':H,'state':ReportState,'revision':I,'createdAt':T}),description='Public allowlist response only. ADMIN, trashed and purged reports return the same neutral 404. Compare SHA-256 of the exact downloaded canonical JSON bytes.')
add('POST','/verify-report','Compare a public USER report verification tuple',obj({'verified':{'const':True},'reportId':U,'revision':I,'canonicalHash':H}),obj({'reportId':U,'revision':I,'canonicalHash':H}),description='Exact report ID, revision and canonical hash comparison; active USER reports only. All absent, mismatching, ADMIN, trashed or purged cases return the same neutral 404.')
add('POST','/exports','Queue a consistent CSV or JSON export',obj({'id':U,'state':{'const':'queued'}}),obj({'kind':ReportKind,'format':enum('csv','json'),'compression':enum('auto','none','gzip','zip')}),202,'Snapshot is taken at worker start; at most 50,000 rows / 16 MiB in a bounded REPEATABLE READ transaction. Limits fail explicitly without partial output. Global concurrency 2, per-owner 1; five-minute aging prevents USER starvation.')
add('GET','/exports','List own export jobs',arr(Export))
add('POST','/exports/{id}/{action}','Cancel or retry an owned export',description='action=cancel or retry. Retained snapshot values are reused on retry; cancelled/expired staging is cleaned. Completed outputs expire after 24 hours.')
add('POST','/artifacts/{id}/credentials','Issue one-use session-bound artifact credentials',obj({'token':S}),code=201,description='Cookie session only; credential expires after five minutes. ADMIN artifacts require current role and fresh reauthentication. Report trash/purge blocks issuing and streaming.')
add('POST','/artifacts/{id}/claim','Claim a credential once for resumable delivery',obj({'url':S}),obj({'token':S}),description='Sets path-scoped HttpOnly filexArtifact cookie. Subsequent Range requests also require the original filexSession. Coverage completion consumes the credential; owner can issue another while artifact is retained.')
for method in ['GET','HEAD']:add(method,'/artifacts/{id}/content','Read encrypted artifact through a fenced Range stream','binary',description='HEAD never consumes; single byte Range supported. Every write rechecks the session, role, reauthentication for admin artifacts, artifact TTL and report state. Full coverage completes this credential.')
add('POST','/views','Save a private transfer view',ID,obj({'name':{'type':'string','minLength':1,'maxLength':100},'filter':Filter}),201)
add('GET','/views','List private saved views',arr(obj({'id':U,'name':S,'filter':Filter})))
add('DELETE','/views/{id}','Delete an owned saved view')
add('POST','/bulk','Freeze a server-side bulk selection',obj({'id':U,'state':enum('paused','queued'),'selected':I}),obj({'ids':{'type':'array','minItems':1,'maxItems':1000,'items':U},'action':enum('revoke','cancel_upload'),'adminScope':B,'reason':Reason,'dryRun':B},['ids','action']),202,'dryRun=true creates a paused selection snapshot. ADMIN scope requires reauthentication/reason; worker checks current role/account epoch. Each selected item is revalidated at execution and reports conflicts.')
add('GET','/bulk','List own bulk jobs and progress',arr(Bulk))
add('POST','/bulk/{id}/{action}','Run or cancel a frozen bulk selection',description='action=run or cancel. A cancelled job stops before its next item; already completed item mutations remain.')
# Route metadata is inspected only for scope wiring, never to invent operation semantics.
sources='\n'.join(p.read_text() for p in (root/'internal/httpapi').glob('*.go') if not p.name.endswith('_test.go'))
paths={}
public={'/api/v1/verify-report','/api/v1/status','/api/v1/banner','/api/v1/auth/login','/api/v1/recipient/session','/api/v1/signing-keys/{id}'}
for (method,path),v in ops.items():
 admin='/admin/' in path
 ispublic=path in public or '/verify/reports/' in path or method=='OPTIONS'
 recipient='/recipient/' in path and not path.endswith('/session')
 secure=[] if ispublic else ([{'recipient':[]}] if recipient else ([{'session':[]}] if admin or path=='/api/v1/events' or '/artifacts/' in path else [{'session':[]},{'bearer':[]}]))
 if '/artifacts/' in path and path.endswith('/content'):secure=[{'session':[],'artifact':[]}]
 match=re.search(r'"'+re.escape(method+' '+path)+r'"[^\n]{0,80}?a.protect\("([^"]+)",\s*(true|false)',sources)
 description=v['description'] or v['title']+'.'
 if admin:description+=' ADMIN cookie session with reauthentication within five minutes is required.'
 op={'operationId':method.lower()+''.join(w.capitalize() for w in re.findall('[a-zA-Z]+',path.removeprefix('/api/v1'))),'summary':v['title'],'description':description,'x-implementation':'implemented','tags':[path.split('/')[3]],'security':secure,'responses':{},'parameters':[]}
 if match:op['x-required-scope']=match[1]
 for name in re.findall(r'{(\w+)}',path):
  schema=U if name not in ('revision','action') else ({'type':'integer','minimum':1} if name=='revision' else enum(*(['cancel','retry'] if '/exports/' in path else ['run','cancel'])))
  op['parameters'].append({'name':name,'in':'path','required':True,'schema':schema,'description':'Resource identifier or allowed action.'})
 if method not in ('GET','HEAD','OPTIONS'):
  op['parameters'].append({'name':'Origin','in':'header','required':(ispublic and path!='/api/v1/verify-report') or recipient,'schema':S,'description':'Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication.'})
  if not ispublic and not recipient:op['parameters'].append({'name':'X-CSRF-Token','in':'header','schema':S,'description':'Required with cookie authentication; obtained at login. Not required for bearer requests.'})
 if v['body'] is not None:op['requestBody']={'required':True,'content':{'application/json':{'schema':v['body']}}}
 if 'forensic' in path:
  form=obj({'reason':Reason,'_csrf':S},['reason','_csrf'])
  if not path.endswith('forensic-zip'):form['properties'].update({'rawCiphertext':enum('true','false'),'objectKey':S})
  op['requestBody']['content']['application/x-www-form-urlencoded']={'schema':form}
 code=str(v['code']);op['responses'][code]={'description':'Success'}
 if v['response']=='binary':
  if method!='HEAD':op['responses'][code]['content']={'application/octet-stream':{'schema':{'type':'string','format':'binary'}}}
  if path.endswith('/content'):
   op['parameters'].append({'name':'Range','in':'header','schema':{'type':'string','pattern':'^bytes=(?:[0-9]+-[0-9]*|-[0-9]+)$'},'description':'One range only; suffix ranges accepted.'})
   op['responses']['206']=copy.deepcopy(op['responses'][code]);op['responses']['206']['description']='Partial content'
   op['responses']['416']={'description':'Unsatisfiable range; Content-Range contains total length.'}
   op['responses'][code]['headers']={'Accept-Ranges':{'schema':{'const':'bytes'}},'Content-Length':{'schema':I},'Content-Disposition':{'schema':S}}
   op['responses']['206']['headers']={'Content-Range':{'schema':S},'Content-Length':{'schema':I},'Accept-Ranges':{'schema':{'const':'bytes'}}}
 elif isinstance(v['response'],dict):op['responses'][code]['content']={'application/json':{'schema':v['response']}}
 if not ispublic:
  for code,desc in [('401','Missing, expired or revoked authentication'),('403','Origin, CSRF, role, reauthentication, ownership or scope rejected'),('404','Resource unavailable'),('409','State, offset, revision, lease or key changed')]:op['responses'][code]={'description':desc,'content':{'application/json':{'schema':Error}}}
 for code,desc in [('400','Invalid input'),('429','Rate or concurrency limit; retry later'),('500','Fail-closed internal failure without private details')]:op['responses'][code]={'description':desc,'content':{'application/json':{'schema':Error}}}
 if path=='/api/v1/status':op['responses']={'200':op['responses']['200'],'503':{'description':'Unavailable','content':{'application/json':{'schema':Status}}}}
 tus=(path.endswith('/uploads') or re.fullmatch(r'/api/v1/uploads/{id}',path))
 if tus:
  if method!='OPTIONS':op['parameters'].append({'name':'Tus-Resumable','in':'header','required':True,'schema':{'const':'1.0.0'}});op['responses']['412']={'description':'Missing/unsupported tus version; Tus-Version: 1.0.0.'}
  for response in op['responses'].values():response.setdefault('headers',{})['Tus-Resumable']={'schema':{'const':'1.0.0'}}
  if method in ['POST','PATCH']:op['parameters'].append({'name':'Upload-Length' if method=='POST' else 'Upload-Offset','in':'header','required':True,'schema':I})
  if method=='POST':op['parameters'].append({'name':'Upload-Metadata','in':'header','required':True,'schema':S,'example':'filename YS50eHQ='});op['responses']['201']['headers']['Location']={'schema':S}
  if method=='PATCH':op['requestBody']={'required':True,'content':{'application/offset+octet-stream':{'schema':{'type':'string','format':'binary','maxLength':4194304}}}};op['responses']['413']={'description':'Chunk larger than 4 MiB'};op['responses']['415']={'description':'Unsupported PATCH media type'}
  for success in ['200','201','204']:
   if success in op['responses']:op['responses'][success]['headers'].update({'Upload-Offset':{'schema':I},'Upload-Length':{'schema':I}})
 if path=='/api/v1/admin/storage/migrations' and method=='POST':op['responses']['200']={'description':'Read-only dry-run totals','content':{'application/json':{'schema':obj({'files':I,'bytes':I})}}}
 if path in ['/api/v1/notifications','/api/v1/events']:op['parameters'].append({'name':'after','in':'query','schema':I,'description':'Last received monotonic notification ID.'})
 if path=='/api/v1/transfers' and method=='GET':op['parameters'].append({'name':'cursor','in':'query','schema':S,'description':'nextCursor from the previous result.'})
 if path.endswith('/qr'):op['parameters'].append({'name':'format','in':'query','schema':enum('png','svg')})
 if not op['parameters']:op.pop('parameters')
 paths.setdefault(path,{})[method.lower()]=op
# Signing API field names are intentionally verified against its implementation.
api={'openapi':'3.1.1','info':{'title':'FileX API','version':'0.1.0','description':'Implemented local-development API for phases 1–5. Generated by scripts/build_contracts.py. Cookie mutations require Origin and X-CSRF-Token; recipient mutations require Origin. Capability, PIN and secrets must never be logged. Production deployment remains gated by phases 6–7.'},'servers':[{'url':'http://localhost:8090','description':'Isolated local development through Traefik'}],'paths':paths,'components':{'schemas':D,'securitySchemes':{'session':{'type':'apiKey','in':'cookie','name':'filexSession'},'recipient':{'type':'apiKey','in':'cookie','name':'filexRecipient'},'artifact':{'type':'apiKey','in':'cookie','name':'filexArtifact'},'bearer':{'type':'http','scheme':'bearer','description':'Scoped API token; no privileged ADMIN operations.'}}}}
outputs={'contracts/openapi.json':api}
for name,definition in [('event-v1',D['OwnerEvent']),('webhook-v1',D['WebhookEvent'])]:outputs['contracts/schemas/'+name+'.json']={'$schema':'https://json-schema.org/draft/2020-12/schema','$id':'https://filex.invalid/schemas/'+name+'.json',**definition}
asyncapi={'asyncapi':'3.0.0','info':{'title':'FileX events','version':'0.1.0','description':'Implemented owner WebSocket notifications and outbound signed webhooks. PostgreSQL outbox is authoritative. At-least-once delivery; consumers persist and deduplicate event IDs.'},'defaultContentType':'application/json','channels':{},'operations':{},'components':{'messages':{}}}
for key,address,schema,description in [('owner','/api/v1/events','event-v1','Cookie and exact Origin; resume with after cursor. Falls back to GET /notifications. Rechecks session every second; no recipient identity.'),('webhook','configured HTTPS endpoint','webhook-v1','Public IPv4 HTTPS:443, no redirect/proxy. X-FileX-Event-ID matches signed eventId; X-FileX-Timestamp within 300s; X-FileX-Signature is hex HMAC-SHA256(secret, timestamp + dot + exact body). Persist eventId atomically with consumer effects. Retry 8 attempts then dead; manual resend retains ID.')]:
 asyncapi['components']['messages'][key]={'name':key,'title':key+' notification','summary':description,'payload':{'schemaFormat':'application/schema+json;version=draft-2020-12','schema':{'$ref':'schemas/'+schema+'.json'}}}
 asyncapi['channels'][key]={'address':address,'description':description,'messages':{key:{'$ref':'#/components/messages/'+key}}}
 asyncapi['operations']['send'+key.title()]={'action':'send','channel':{'$ref':'#/channels/'+key},'x-implementation':'implemented','messages':[{'$ref':'#/channels/'+key+'/messages/'+key}]}
outputs['contracts/asyncapi.json']=asyncapi
for path,value in outputs.items():
 result=json.dumps(value,ensure_ascii=False,indent=2)+'\n';target=root/path
 if '--check' in sys.argv:
  if not target.exists() or target.read_text()!=result:raise SystemExit('Contract drift: run python3 scripts/build_contracts.py: '+path)
 else:target.write_text(result)
print(f'{len(ops)} implemented operations; OpenAPI, AsyncAPI and event schemas synchronized.')
