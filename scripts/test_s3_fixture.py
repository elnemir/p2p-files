"""Validate fixture key boundaries and the wire operations consumed by the load harness."""
from http.client import HTTPConnection
from http.server import ThreadingHTTPServer
from pathlib import Path
import tempfile
import threading
import unittest
from unittest.mock import patch
from uuid import UUID
from xml.etree import ElementTree

from fixtures import s3


class S3FixtureTests(unittest.TestCase):
    def test_request_values_cannot_select_paths_or_noncanonical_keys(self):
        valid = '12345678-1234-4abc-8def-1234567890ab'
        self.assertEqual(s3.ROOT / valid, s3.object_path(valid))
        for key in ['../secret', '/etc/passwd', '..\\secret', '%2e%2e%2fsecret',
                    valid + '/extra', valid + '?q=x', valid.upper(), valid.replace('-', ''),
                    '{' + valid + '}', '-' * 36, None, '']:
            with self.subTest(key=key):
                self.assertIsNone(s3.object_path(key))

    def test_wire_lifecycle_and_invalid_key_have_no_filesystem_side_effect(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(s3, 'ROOT', Path(directory)):
            server = ThreadingHTTPServer(('127.0.0.1', 0), s3.Handler)
            worker = threading.Thread(target=server.serve_forever, daemon=True)
            worker.start()

            def request(method, path, body=None, **headers):
                connection = HTTPConnection('127.0.0.1', server.server_port, timeout=5)
                try:
                    connection.request(method, path, body, {'Authorization': 'AWS4-HMAC-SHA256 fixture',
                        'Connection': 'close', **headers})
                    response = connection.getresponse()
                    return response.status, response.read()
                finally:
                    connection.close()

            try:
                for key in ['../escape', '%2e%2e%2fescape', '-' * 36, '12345678-1234-4ABC-8DEF-1234567890AB']:
                    self.assertEqual(400, request('PUT', '/fixture/' + key, b'blocked', **{'If-None-Match': '*'})[0])
                self.assertEqual([], list(Path(directory).iterdir()))
                key = str(UUID(int=10))
                endpoint = '/fixture/' + key
                self.assertEqual(200, request('PUT', endpoint, b'fixture payload', **{'If-None-Match': '*'})[0])
                self.assertEqual(412, request('PUT', endpoint, b'overwrite', **{'If-None-Match': '*'})[0])
                self.assertEqual((200, b'fixture payload'), request('GET', endpoint))
                self.assertEqual(200, request('HEAD', endpoint)[0])
                status, body = request('GET', '/fixture?list-type=2')
                self.assertEqual(200, status)
                self.assertEqual(key, ElementTree.fromstring(body).find('.//{*}Key').text)
                self.assertEqual(204, request('DELETE', endpoint)[0])
                self.assertEqual(404, request('GET', endpoint)[0])
                self.assertEqual([], list(Path(directory).iterdir()))
            finally:
                server.shutdown()
                server.server_close()
                worker.join(timeout=5)


if __name__ == '__main__':
    unittest.main()
