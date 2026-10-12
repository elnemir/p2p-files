import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile,readdir} from 'node:fs/promises';
import Ajv from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';
const load=async p=>JSON.parse(await readFile(p,'utf8'));
const ajv=new Ajv({strict:true});addFormats(ajv);
const event=ajv.compile(await load('contracts/schemas/event-v1.json'));
const hook=ajv.compile(await load('contracts/schemas/webhook-v1.json'));
const manifest=ajv.compile(await load('contracts/schemas/manifest-v1.json'));
const davCredential=ajv.compile(await load('contracts/schemas/webdav-credential-v1.json'));
const id='00000000-0000-4000-8000-000000000001';
test('owner events and webhooks exclude recipient correlation',()=>{
 const valid={schemaVersion:1,id:1,kind:'transfer.published',transferId:id,createdAt:'2026-10-11T00:00:00Z',readAt:null};assert.equal(event(valid),true);assert.equal(event({...valid,recipientEmail:'private@example.test'}),false);assert.equal(event({...valid,createdAt:'yesterday'}),false);
 const webhook={schemaVersion:1,eventId:id,type:'download.completed',transferId:id};assert.equal(hook(webhook),true);assert.equal(hook({...webhook,recipientEmail:'private@example.test'}),false);
});
test('manifest requires server checksum and nonempty file set',()=>{
 assert.equal(manifest({version:1,files:[]}),false);assert.equal(manifest({version:1,files:[{id,path:'a.txt',size:0,sha256:'a'.repeat(64)}]}),true);assert.equal(manifest({version:1,files:[{id,path:'a.txt',size:-1,sha256:'wrong'}]}),false);
});
test('every implemented API route is documented and privileged routes require sessions',async()=>{
 const api=await load('contracts/openapi.json');const actual=new Set();for(const file of await readdir('internal/httpapi')){if(!file.endsWith('.go')||file.endsWith('_test.go'))continue;const text=await readFile('internal/httpapi/'+file,'utf8');for(const match of text.matchAll(/Handle(?:Func)?\("(GET|POST|PATCH|PUT|DELETE|HEAD|OPTIONS) (\/api\/v1[^" ]*)"/g))actual.add(match[1]+' '+match[2]);}
 actual.add('OPTIONS /api/v1/uploads/{id}'); // tus middleware dispatches resource OPTIONS.
 const documented=new Set();for(const [path,item]of Object.entries(api.paths))for(const [method,op]of Object.entries(item)){documented.add(method.toUpperCase()+' '+path);assert.equal(op['x-implementation'],'implemented');assert.ok(op.description.length>10);assert.ok(op.responses);if(path.includes('/admin/'))assert.deepEqual(op.security,[{session:[]}]);}
 assert.deepEqual([...documented].sort(),[...actual].sort());
});
test('binary Range and tus recovery contracts remain explicit',async()=>{
 const api=await load('contracts/openapi.json');for(const path of ['/api/v1/recipient/downloads/{id}/content','/api/v1/artifacts/{id}/content']){assert.ok(api.paths[path].get.responses['206']);assert.ok(api.paths[path].get.responses['416']);assert.equal(api.paths[path].head.responses['200'].content,undefined);}
 const patch=api.paths['/api/v1/uploads/{id}'].patch;assert.ok(patch.responses['412']);assert.ok(patch.responses['409']);assert.ok(patch.requestBody.content['application/offset+octet-stream']);
});
test('WebDAV has an explicit non-REST protocol contract and scoped credential schema',async()=>{
 const api=await load('contracts/openapi.json');const dav=await load('contracts/webdav.json');assert.deepEqual(dav,api['x-webdav']);
 for(const method of ['OPTIONS','PROPFIND','HEAD','PUT','MKCOL','MOVE','DELETE','PROPPATCH','GET','COPY','LOCK','UNLOCK'])assert.ok(dav.methods[method]);
 assert.equal(dav.limits.chunkBytes,4194304);assert.equal(dav.limits.maxXMLBytes,65536);assert.equal(dav.limits.maxDepth,1);
 const valid={id,url:'https://filex.example/api/v1/webdav/'+id+'/',username:'filex',password:'fixture',expiresAt:'2026-10-13T00:00:00Z'};
 assert.equal(davCredential(valid),true);assert.equal(davCredential({...valid,username:'admin'}),false);assert.equal(davCredential({...valid,secretHash:'private'}),false);
});
