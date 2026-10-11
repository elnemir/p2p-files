import SwaggerParser from '@apidevtools/swagger-parser';
import Ajv from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';
import type {Page} from '@playwright/test';
export async function contractObserver(){
 const api:any=await SwaggerParser.dereference('contracts/openapi.json');
 const normalize=(value:any):any=>{if(Array.isArray(value))return value.map(normalize);if(value&&typeof value==='object')return Object.fromEntries(Object.entries(value).filter(([k])=>k!=='$id'&&k!=='$schema').map(([k,v])=>[k,normalize(v)]));return value;};
 const ajv=new Ajv({strict:false});addFormats(ajv);const checks=new Map<string,any>();const failures:string[]=[];const pending:Promise<void>[]=[];let count=0;
 function observe(page:Page){page.on('requestfinished',request=>{const job=(async()=>{const response=await request.response();if(!response)return;const url=new URL(response.url());if(url.pathname.endsWith('/content')||!url.pathname.startsWith('/api/v1/')||!response.headers()['content-type']?.includes('application/json'))return;const path=Object.keys(api.paths).find(path=>new RegExp('^'+path.replace(/\{[^}]+\}/g,'[^/]+')+'$').test(url.pathname));const method=response.request().method().toLowerCase();const op=path&&api.paths[path][method];const schema=op?.responses[String(response.status())]?.content?.['application/json']?.schema;if(!schema){failures.push(`${method} ${path||url.pathname}: undocumented JSON status ${response.status()}`);return;}const key=method+path+response.status();if(!checks.has(key))checks.set(key,ajv.compile(normalize(schema)));let body;try{body=await response.json()}catch{return}const validate=checks.get(key);count++;if(!validate(body))failures.push(`${method} ${path}: ${JSON.stringify(validate.errors)}`);})();pending.push(job);});}
 return {observe,async finish(){await Promise.all(pending);return {failures,count};}};
}
