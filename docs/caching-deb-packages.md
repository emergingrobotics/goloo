# Caching Deb Packages Locally for Multipass VMs

apt-cacher-ng is a caching proxy for Debian/Ubuntu packages. It runs on the host machine and caches packages locally. The first VM downloads packages normally from upstream; every subsequent VM gets them from the local cache.

## Setup

### Linux host

```bash
sudo apt install apt-cacher-ng
# Runs on port 3142 by default
```

### macOS host (Docker)

```bash
docker run -d --name apt-cacher-ng -p 3142:3142 \
  -v apt-cacher-ng-data:/var/cache/apt-cacher-ng \
  --restart unless-stopped \
  sameersbn/apt-cacher-ng
```

## Pointing VMs at the cache

In cloud-init, add an `apt` section that sets the proxy to your host's IP. Multipass VMs can reach the host at the bridge IP (typically `192.168.64.1` on macOS, check with `ip route` inside the VM):

```yaml
#cloud-config
apt:
  proxy: http://192.168.64.1:3142
```

Or with Go templates, make it configurable:

```yaml
#cloud-config
{{- if .Vars.apt_proxy}}
apt:
  proxy: {{.Vars.apt_proxy}}
{{- end}}
```

With the corresponding config.json:

```json
{
  "cloud_init": {
    "vars": {
      "apt_proxy": "http://192.168.64.1:3142"
    }
  }
}
```

## How it works

- VMs request packages through the proxy instead of directly from archive.ubuntu.com
- First request: apt-cacher-ng fetches from upstream, caches locally, serves to VM
- Subsequent requests: served directly from cache
- Cache survives VM create/delete cycles

A typical cloud-init with 15-20 packages goes from minutes to seconds on the second run.
