"""Render a standalone immutable-image ingress configuration without running it."""
import ipaddress
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]


def configuration(domain, email, subnet, api_image, web_image, postgres_image, redis_image, traefik_image):
    if len(domain) > 253 or domain != domain.lower() or len(domain.split('.')) < 2 or any(not re.fullmatch(r'[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?', label) for label in domain.split('.')):
        raise ValueError('domain must be a lower-case DNS name')
    try:
        ipaddress.ip_address(domain)
    except ValueError:
        pass
    else:
        raise ValueError('ACME domain cannot be an IP address')
    if not re.fullmatch(r'[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+', email):
        raise ValueError('valid ACME contact email required')
    network = ipaddress.IPv4Network(subnet, strict=True)
    private = [ipaddress.IPv4Network(c) for c in ('10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16')]
    if not any(network.subnet_of(c) for c in private) or not 24 <= network.prefixlen <= 29:
        raise ValueError('proxy subnet must be RFC1918 /24 through /29; check host overlap separately')
    for name in (api_image, web_image, postgres_image, redis_image, traefik_image):
        if not re.fullmatch(r'[a-z0-9][a-z0-9._:/-]*@sha256:[a-f0-9]{64}', name):
            raise ValueError('runtime image must have an immutable SHA-256 digest')
    proxy_ip = str(network.network_address + 2)
    api_ip = str(network.network_address + 3)
    web_ip = str(network.network_address + 4)
    hardening = {'read_only': True, 'cap_drop': ['ALL'], 'security_opt': ['no-new-privileges:true'], 'pids_limit': 128, 'restart': 'unless-stopped', 'logging': {'driver': 'json-file', 'options': {'max-size': '10m', 'max-file': '3'}}}
    database_env = {'APP_ENV': 'production', 'DATABASE_URL': 'postgres://filex:${POSTGRES_PASSWORD:?Set a URL-safe random password}@postgres:5432/filex?sslmode=disable', 'MASTER_KEK': '${MASTER_KEK:?Set an independently generated base64 32-byte KEK}'}
    data_mounts = ['payload_data:/data/payload', 'artifact_data:/data/artifacts', 'backup_data:/data/backups']
    services = {
        'postgres': {**hardening, 'image': postgres_image, 'user': '70:70', 'environment': {'POSTGRES_USER': 'filex', 'POSTGRES_PASSWORD': '${POSTGRES_PASSWORD:?Set a URL-safe random password}', 'POSTGRES_DB': 'filex'}, 'volumes': ['pg_data:/var/lib/postgresql'], 'tmpfs': ['/tmp', '/var/run/postgresql:uid=70,gid=70,mode=0750'], 'networks': ['backend'], 'mem_limit': '2g', 'cpus': 2, 'healthcheck': {'test': ['CMD-SHELL', 'pg_isready -U filex -d filex'], 'interval': '3s', 'timeout': '3s', 'retries': 20}},
        'redis': {**hardening, 'image': redis_image, 'user': '999:999', 'command': ['redis-server', '--save', '', '--appendonly', 'no', '--maxmemory', '128mb', '--maxmemory-policy', 'allkeys-lru'], 'tmpfs': ['/data:uid=999,gid=999,mode=0700'], 'networks': ['backend'], 'mem_limit': '192m', 'cpus': 0.5},
        'api': {**hardening, 'image': api_image, 'user': '65532:65532', 'environment': {**database_env, 'HTTP_ADDR': '0.0.0.0:8080', 'INTERNAL_ADDR': '127.0.0.1:9090'}, 'depends_on': {'postgres': {'condition': 'service_healthy'}}, 'volumes': data_mounts, 'networks': {'proxy': {'ipv4_address': api_ip}, 'backend': {}, 'egress': {}}, 'mem_limit': '768m', 'cpus': 2},
        'worker': {**hardening, 'image': api_image, 'user': '65532:65532', 'entrypoint': ['filex-worker'], 'environment': database_env, 'depends_on': {'postgres': {'condition': 'service_healthy'}}, 'volumes': data_mounts, 'networks': ['backend', 'egress'], 'mem_limit': '768m', 'cpus': 2},
        'web': {**hardening, 'image': web_image, 'user': '1000:1000', 'environment': {'HOSTNAME': '0.0.0.0', 'PORT': '3000', 'FILEX_BRANDING_API_URL': 'http://api:9090/branding', 'NEXT_TELEMETRY_DISABLED': '1'}, 'tmpfs': ['/tmp'], 'networks': {'proxy': {'ipv4_address': web_ip}}, 'mem_limit': '512m', 'cpus': 1},
        'traefik': {**hardening, 'image': traefik_image, 'user': '65532:65532', 'command': ['--configFile=/etc/traefik/static.yaml'], 'volumes': ['./traefik-static.yaml:/etc/traefik/static.yaml:ro', './routes.yaml:/etc/traefik/routes.yaml:ro', './acme:/var/lib/traefik'], 'ports': ['80:8080', '443:8443'], 'networks': {'proxy': {'ipv4_address': proxy_ip}, 'egress': {}}, 'depends_on': ['api', 'web'], 'mem_limit': '256m', 'cpus': 1},
    }
    compose = {'name': 'filex-production', 'services': services, 'networks': {'proxy': {'internal': True, 'ipam': {'config': [{'subnet': str(network)}]}}, 'backend': {'internal': True}, 'egress': {}}, 'volumes': {k: {} for k in ['pg_data', 'payload_data', 'artifact_data', 'backup_data']}}
    static = {
        'global': {'checkNewVersion': False, 'sendAnonymousUsage': False},
        'entryPoints': {name: {'address': address, 'forwardedHeaders': {'insecure': False, 'trustedIPs': []}, 'transport': {'respondingTimeouts': {'readTimeout': '60s', 'writeTimeout': '0s', 'idleTimeout': '30s'}}} for name, address in [('web', ':8080'), ('websecure', ':8443')]},
        'providers': {'file': {'filename': '/etc/traefik/routes.yaml', 'watch': True}},
        'api': {'dashboard': False}, 'log': {'level': 'ERROR'},
        'certificatesResolvers': {'letsencrypt': {'acme': {'email': email, 'storage': '/var/lib/traefik/acme.json', 'httpChallenge': {'entryPoint': 'web'}}}},
    }
    host_rule = 'Host(`' + domain + '`)'
    # Literal proxy-network addresses avoid service DNS selecting the shared egress interface.
    routes = {'http': {
        'routers': {
            'redirect': {'entryPoints': ['web'], 'rule': host_rule, 'middlewares': ['https-redirect'], 'service': 'noop@internal'},
            'api': {'entryPoints': ['websecure'], 'rule': host_rule + ' && PathPrefix(`/api/v1/`)', 'priority': 100, 'service': 'api', 'middlewares': ['privacy'], 'tls': {'certResolver': 'letsencrypt'}},
            'web': {'entryPoints': ['websecure'], 'rule': host_rule, 'service': 'web', 'middlewares': ['privacy'], 'tls': {'certResolver': 'letsencrypt'}},
        },
        'middlewares': {'https-redirect': {'redirectScheme': {'scheme': 'https', 'port': '443', 'permanent': True}}, 'privacy': {'headers': {'stsSeconds': 31536000, 'stsIncludeSubdomains': False, 'stsPreload': False, 'contentTypeNosniff': True, 'frameDeny': True, 'referrerPolicy': 'no-referrer', 'customResponseHeaders': {'Cache-Control': 'no-store'}}}},
        'services': {'api': {'loadBalancer': {'servers': [{'url': 'http://' + api_ip + ':8080'}]}}, 'web': {'loadBalancer': {'servers': [{'url': 'http://' + web_ip + ':3000'}]}}},
    }, 'tls': {'options': {'default': {'minVersion': 'VersionTLS12', 'sniStrict': True}}}}
    settings = {'publicOrigin': 'https://' + domain, 'networkPolicy': {'trustedProxyCidrs': [proxy_ip + '/32'], 'allowedClientCidrs': [], 'requireProxy': True, 'requireHttps': True}}
    return {'compose.json': compose, 'traefik-static.yaml': static, 'routes.yaml': routes, 'settings-ingress.json': settings}


def write_configuration(destination, files):
    destination = Path(destination)
    destination.mkdir(mode=0o700, parents=True, exist_ok=False)
    for name, value in files.items():
        with (destination / name).open('x') as f:
            json.dump(value, f, ensure_ascii=False, indent=2)
            f.write('\n')
    (destination / 'runtime.env.example').write_text('POSTGRES_PASSWORD=\nMASTER_KEK=\n')


def main():
    import argparse
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['domain', 'email', 'subnet', 'api-image', 'web-image', 'postgres-image', 'redis-image', 'traefik-image', 'output']:
        parser.add_argument('--' + name, required=True)
    args = parser.parse_args()
    try:
        files = configuration(args.domain, args.email, args.subnet, args.api_image, args.web_image, args.postgres_image, args.redis_image, args.traefik_image)
        write_configuration(args.output, files)
    except (ValueError, OSError) as exc:
        parser.exit(2, 'Configuration rejected: ' + str(exc) + '\n')
    print('Configuration generated. Nothing was deployed. APP_ENV=production remains blocked until release acceptance. See wiki/production-ingress.md.')


if __name__ == '__main__':
    main()
