import tempfile
import unittest
from pathlib import Path

from production_config import configuration, write_configuration


class ProductionConfigurationTest(unittest.TestCase):
    def args(self):
        return dict(domain='files.example.test', email='operator@example.test', subnet='172.30.90.0/29', api_image='registry.example.test/filex-api@sha256:' + 'a' * 64, web_image='registry.example.test/filex-web@sha256:' + 'b' * 64)

    def test_rejects_unsafe_configuration(self):
        for key, value in [('domain', 'files.example.test` || PathPrefix(`/`)'), ('domain', '127.0.0.1'), ('email', 'bad'), ('subnet', '0.0.0.0/0'), ('subnet', '203.0.113.0/24'), ('subnet', '172.30.90.1/29'), ('api_image', 'filex:latest')]:
            with self.subTest(key=key, value=value):
                args = self.args()
                args[key] = value
                with self.assertRaises(ValueError):
                    configuration(**args)

    def test_private_backend_and_immutable_nonroot_services(self):
        files = configuration(**self.args())
        compose = files['compose.json']
        for name, service in compose['services'].items():
            self.assertNotEqual(service['user'].split(':')[0], '0', name)
            self.assertTrue(service['read_only'], name)
            self.assertEqual(service['cap_drop'], ['ALL'], name)
            self.assertIn('@sha256:', service['image'], name)
            self.assertNotIn('build', service, name)
            if name != 'traefik':
                self.assertNotIn('ports', service, name)
            self.assertNotIn('docker.sock', str(service))
        self.assertTrue(compose['networks']['backend']['internal'])
        self.assertTrue(compose['networks']['proxy']['internal'])
        self.assertEqual(compose['services']['api']['environment']['APP_ENV'], 'production')
        self.assertEqual(compose['services']['api']['environment']['INTERNAL_ADDR'], '127.0.0.1:9090')
        self.assertEqual(files['settings-ingress.json']['networkPolicy']['trustedProxyCidrs'], ['172.30.90.2/32'])
        self.assertEqual(files['routes.yaml']['http']['services']['api']['loadBalancer']['servers'][0]['url'], 'http://172.30.90.3:8080')
        self.assertEqual(files['routes.yaml']['tls']['options']['default'], {'minVersion': 'VersionTLS12', 'sniStrict': True})

    def test_never_overwrites_existing_output(self):
        with tempfile.TemporaryDirectory() as directory:
            out = Path(directory) / 'generated'
            files = configuration(**self.args())
            write_configuration(out, files)
            (out / 'runtime.env.example').write_text('operator-owned\n')
            with self.assertRaises(FileExistsError):
                write_configuration(out, files)
            self.assertEqual((out / 'runtime.env.example').read_text(), 'operator-owned\n')


if __name__ == '__main__':
    unittest.main()
