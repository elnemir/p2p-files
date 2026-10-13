# Test client only; never shipped as a FileX service. Package versions are recorded
# by the native check because Ubuntu security updates intentionally remain enabled.
FROM ubuntu:24.04@sha256:534baea6a22c03a63003dbc8dbe78fe34bc0d7e595d9a9dc9834884ff530eb55
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends libglib2.0-bin gvfs-backends gvfs-daemons dbus-x11 python3 ca-certificates && rm -rf /var/lib/apt/lists/*
