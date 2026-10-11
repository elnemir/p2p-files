"""Disposable TLS S3 wire fixture, never a production object store.

Implements only the operations used by FileX. It checks the signing envelope,
not cryptographic SigV4 correctness; real provider acceptance remains separate.
"""
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import os,re,ssl,threading
from urllib.parse import urlsplit,parse_qs
from xml.sax.saxutils import escape

ROOT=Path('/data');ROOT.mkdir(exist_ok=True)
LOCK=threading.Lock()

class Handler(BaseHTTPRequestHandler):
    protocol_version='HTTP/1.1'
    def log_message(self,*args):pass
    def reply(self,status,body=b'',content='application/octet-stream'):
        self.send_response(status);self.send_header('Content-Length',str(len(body)));self.send_header('Content-Type',content);self.end_headers()
        if self.command!='HEAD':self.wfile.write(body)
    def dispatch(self):
        if not self.headers.get('Authorization','').startswith('AWS4-HMAC-SHA256 '):return self.reply(403)
        url=urlsplit(self.path);parts=url.path.strip('/').split('/')
        if parts[0]!='fixture':return self.reply(404)
        with LOCK:
            if len(parts)==1:
                if self.command=='HEAD':return self.reply(200)
                if self.command!='GET':return self.reply(405)
                query=parse_qs(url.query)
                after=query.get('continuation-token',[''])[0]
                limit=max(1,min(1000,int(query.get('max-keys',['1000'])[0])))
                names=sorted(p.name for p in ROOT.iterdir() if p.is_file() and p.name>after)
                batch=names[:limit]
                records=''.join('<Contents><Key>'+escape(n)+'</Key><Size>'+str((ROOT/n).stat().st_size)+'</Size></Contents>' for n in batch)
                more=len(names)>limit
                result='<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>'+str(more).lower()+'</IsTruncated>'+records+('<NextContinuationToken>'+batch[-1]+'</NextContinuationToken>' if more else '')+'</ListBucketResult>'
                return self.reply(200,result.encode(),'application/xml')
            if len(parts)!=2 or not re.fullmatch('[a-f0-9-]{36}',parts[1]):return self.reply(400)
            file=ROOT/parts[1]
            if self.command=='PUT':
                size=int(self.headers.get('Content-Length','-1'))
                if size<0 or size>4194816:return self.reply(413)
                if file.exists():return self.reply(412)
                if self.headers.get('If-None-Match')!='*':return self.reply(400)
                data=self.rfile.read(size)
                if len(data)!=size:return self.reply(400)
                file.write_bytes(data);return self.reply(200)
            if self.command=='DELETE':file.unlink(missing_ok=True);return self.reply(204)
            if self.command in ['GET','HEAD']:
                if not file.exists():return self.reply(404,b'<Error><Code>NoSuchKey</Code></Error>','application/xml')
                return self.reply(200,file.read_bytes())
            return self.reply(405)
    do_GET=do_HEAD=do_PUT=do_DELETE=dispatch

server=ThreadingHTTPServer(('0.0.0.0',8443),Handler)
context=ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
context.minimum_version=ssl.TLSVersion.TLSv1_2
context.load_cert_chain('/fixtures/server.crt','/fixtures/server.key')
server.socket=context.wrap_socket(server.socket,server_side=True)
server.serve_forever()
