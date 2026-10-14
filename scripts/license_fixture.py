"""Ephemeral signed licenses for isolated local qualification, never release trust."""
import json
import subprocess

NODE = r'''
const crypto=require('node:crypto');
function key(){const k=crypto.generateKeyPairSync('ed25519');return {private:k.privateKey.export({type:'pkcs8',format:'pem'}),public:k.publicKey.export({type:'spki',format:'der'}).subarray(-32).toString('base64')}}
function sign(value,type,k){const h=Buffer.from(JSON.stringify({alg:'EdDSA',kid:'qualification-only',typ:type})).toString('base64url');const p=Buffer.from(JSON.stringify(value)).toString('base64url');const data=h+'.'+p;return data+'.'+crypto.sign(null,Buffer.from(data),k).toString('base64url')}
let input='';process.stdin.on('data',b=>input+=b);process.stdin.on('end',()=>{const v=JSON.parse(input);if(v.action==='keys'){const license=key(),release=key(),source=crypto.randomBytes(32).toString('hex');console.log(JSON.stringify({private:license.private,env:{FILEX_LICENSE_TRUST:JSON.stringify({'qualification-only':license.public}),FILEX_RELEASE_TRUST:JSON.stringify({'qualification-only':release.public}),FILEX_RELEASE_SOURCE:source,FILEX_RELEASE_METADATA:sign({version:'qualification',sourceSha256:source,publishedAt:new Date(Date.now()-3600000).toISOString()},'filex-release+jws',release.private)}}))}else{const now=new Date(Date.now()-60000).toISOString(),end=new Date(Date.now()+86400000).toISOString();console.log(JSON.stringify({envelope:sign({schemaVersion:1,product:'filex',issuer:'filex-vendor',licenseId:crypto.randomUUID(),entitlementId:crypto.randomUUID(),revision:1,customerId:'qualification-only',installationId:v.installationId,kind:'subscription',issuedAt:now,notBefore:now,expiresAt:end,updatesUntil:end,supportUntil:end,limits:{maxEnabledUsers:null,maxTransferBytes:null},features:['webdav','federated_auth','white_label']},'filex-license+jws',v.private),expectedEpoch:v.epoch}))}})
'''
def call(value):
    return json.loads(subprocess.run(['node','-e',NODE],input=json.dumps(value),text=True,capture_output=True,check=True).stdout)

def new_fixture():
    return call({'action':'keys'})

def activate(compose, fixture):
    status=json.loads(compose('exec','-T','api','filex-cli','license','status',capture_output=True).stdout)
    value=call({'action':'license','private':fixture['private'],'installationId':status['installationId'],'epoch':status['epoch']})
    compose('exec','-T','api','filex-cli','license','apply',input=json.dumps(value),stdout=subprocess.DEVNULL)
    return value

def renewal(fixture, previous):
    code = r"""
const crypto=require('node:crypto');let input='';process.stdin.on('data',v=>input+=v);process.stdin.on('end',()=>{const v=JSON.parse(input),parts=v.envelope.split('.'),c=JSON.parse(Buffer.from(parts[1],'base64url'));c.revision++;c.licenseId=crypto.randomUUID();const data=parts[0]+'.'+Buffer.from(JSON.stringify(c)).toString('base64url');console.log(JSON.stringify({envelope:data+'.'+crypto.sign(null,Buffer.from(data),v.private).toString('base64url')}))})
"""
    return json.loads(subprocess.run(['node','-e',code],input=json.dumps({'private':fixture['private'],'envelope':previous['envelope']}),text=True,capture_output=True,check=True).stdout)['envelope']
