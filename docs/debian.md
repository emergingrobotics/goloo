# Debian Packaging for Goloo

How to build `.deb` packages, host an APT repository, and produce multi-architecture builds.

## What's Inside a .deb File

A `.deb` is an `ar` archive containing three members:

```
goloo_1.0.0-1_amd64.deb (ar archive)
├── debian-binary        # Format version ("2.0")
├── control.tar.xz       # Package metadata
└── data.tar.xz          # Installed files
```

The `control.tar` contains the `DEBIAN/control` file (package name, version, architecture, dependencies, description) plus optional maintainer scripts (`preinst`, `postinst`, `prerm`, `postrm`) and `md5sums`.

The `data.tar` contains the actual files rooted at `/`:

```
./usr/bin/goloo
./usr/share/doc/goloo/copyright
./usr/share/doc/goloo/changelog.Debian.gz
```

Debian policy: binaries go in `/usr/bin/`, man pages in `/usr/share/man/man1/`, docs in `/usr/share/doc/<package>/`.

## The DEBIAN/control File

```
Package: goloo
Version: 1.0.0-1
Section: admin
Priority: optional
Architecture: amd64
Maintainer: Your Name <you@example.com>
Description: VM provisioning CLI
 Provisions local and AWS virtual machines using
 identical configuration files.
Homepage: https://github.com/emergingrobotics/goloo
```

Key fields:

- **Version**: `[epoch:]upstream_version[-debian_revision]`. For goloo 1.0.0 with first packaging: `1.0.0-1`.
- **Architecture**: `amd64`, `arm64`, `armhf`, `i386`, or `all` (arch-independent).
- **Depends**: For a `CGO_ENABLED=0` Go binary, this can be empty (fully static, zero library dependencies).
- **Description**: First line is a short synopsis (< 80 chars). Subsequent lines are indented by one space. A ` .` line creates a paragraph break.

Filename convention: `{package}_{version}_{architecture}.deb`

## Building .deb Packages

### Method 1: Manual Build with dpkg-deb

The fast approach for CI pipelines and internal tooling.

```bash
# Create directory structure
mkdir -p build-deb/DEBIAN build-deb/usr/bin

# Build the Go binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build-deb/usr/bin/goloo ./cmd/goloo/

# Write the control file
cat > build-deb/DEBIAN/control <<EOF
Package: goloo
Version: 1.0.0-1
Architecture: amd64
Maintainer: Your Name <you@example.com>
Description: VM provisioning CLI
 Provisions local and AWS virtual machines.
EOF

# Fix permissions
chmod 755 build-deb/usr/bin/goloo

# Generate md5sums
(cd build-deb && find usr -type f -exec md5sum {} \; > DEBIAN/md5sums)

# Build the .deb
dpkg-deb --build build-deb goloo_1.0.0-1_amd64.deb
```

Note: `DEBIAN/` (uppercase) is used in binary packages and when building with `dpkg-deb`. This is different from `debian/` (lowercase) used in source packages.

### Method 2: Proper Debian Source Package (debhelper)

Required for official Debian/Ubuntu archives.

A `debian/` directory lives inside the source tree:

```
goloo/
├── cmd/
├── internal/
├── go.mod
├── debian/
│   ├── control          # Source + binary package metadata
│   ├── rules            # Build instructions (Makefile)
│   ├── changelog        # Determines package version
│   ├── copyright        # License info (DEP-5 machine-readable format)
│   └── source/
│       └── format       # "3.0 (quilt)" for upstream projects
```

**debian/control** (source package version):

```
Source: goloo
Section: admin
Priority: optional
Maintainer: Your Name <you@example.com>
Build-Depends: debhelper-compat (= 13),
               dh-golang,
               golang-any (>= 2:1.21~)
Standards-Version: 4.6.2
Rules-Requires-Root: no
XS-Go-Import-Path: github.com/emergingrobotics/goloo

Package: goloo
Architecture: any
Depends: ${misc:Depends}, ${shlibs:Depends}
Description: VM provisioning CLI tool
 Goloo provisions virtual machines using Multipass locally
 or AWS CloudFormation in the cloud.
```

`Architecture: any` means "build a separate binary for each supported architecture." The build system fills in the correct value for each build.

**debian/rules** (a Makefile, must use tabs):

```makefile
#!/usr/bin/make -f

%:
	dh $@ --buildsystem=golang --with=golang

override_dh_auto_build:
	dh_auto_build -- -ldflags "-X main.version=$(shell dpkg-parsechangelog -S Version)"
```

**debian/changelog** (strictly formatted, use `dch` to edit):

```
goloo (1.0.0-1) unstable; urgency=medium

  * Initial release. (Closes: #1234567)

 -- Your Name <you@example.com>  Sat, 14 Feb 2026 12:00:00 +0000
```

**Build the package:**

```bash
sudo apt install devscripts build-essential dh-golang golang-go

dpkg-buildpackage -us -uc    # unsigned, for testing
dpkg-buildpackage             # signed (needs GPG key)
```

This produces `goloo_1.0.0-1_amd64.deb`, `.dsc`, `.orig.tar.gz`, `.debian.tar.xz`, `.changes`, and `.buildinfo` in the parent directory.

**Lint with:**

```bash
lintian goloo_1.0.0-1_amd64.changes
```

### Method 3: GoReleaser with nfpm (Recommended for Goloo)

GoReleaser handles cross-compilation and multi-arch `.deb` creation in one step with zero Debian tooling dependencies.

**.goreleaser.yaml:**

```yaml
project_name: goloo

builds:
  - id: goloo
    main: ./cmd/goloo/
    binary: goloo
    env:
      - CGO_ENABLED=0
    goos:
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}

nfpms:
  - id: goloo-deb
    package_name: goloo
    vendor: Emerging Robotics
    homepage: https://github.com/emergingrobotics/goloo
    maintainer: Your Name <you@example.com>
    description: |-
      CLI tool for provisioning VMs locally and in AWS.
      Goloo uses identical config files for local Multipass VMs
      and AWS EC2 instances.
    license: MIT
    formats:
      - deb
    recommends:
      - multipass
    contents:
      - src: ./configs/
        dst: /usr/share/goloo/configs/
```

```bash
# Test locally
goreleaser release --snapshot --clean

# Release (tag-triggered)
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
goreleaser release --clean
```

Produces per architecture:

```
dist/
├── goloo_1.0.0_amd64.deb
├── goloo_1.0.0_arm64.deb
├── goloo_1.0.0_linux_amd64.tar.gz
├── goloo_1.0.0_linux_arm64.tar.gz
└── checksums.txt
```

nfpm can also be used standalone without GoReleaser:

```bash
nfpm package --packager deb --target goloo_1.0.0_amd64.deb
```

## Multi-Architecture Builds (ARM + x86)

### Go Cross-Compilation

Go has built-in cross-compilation via `GOOS` and `GOARCH`:

```bash
# x86_64 / amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o goloo-amd64 ./cmd/goloo/

# ARM64 / aarch64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o goloo-arm64 ./cmd/goloo/

# 32-bit ARM (Raspberry Pi 2+)
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o goloo-armv7 ./cmd/goloo/
```

`CGO_ENABLED=0` produces a fully static binary with no C library dependencies. If CGO is needed for cross-compilation, install a cross-compiler toolchain:

```bash
sudo apt install gcc-aarch64-linux-gnu
CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build ...
```

### Architecture Mapping

| Go (GOARCH)    | Debian Architecture | Common Name                       |
|----------------|---------------------|-----------------------------------|
| `amd64`        | `amd64`             | x86_64 / Intel/AMD 64-bit        |
| `arm64`        | `arm64`             | aarch64 / ARMv8 64-bit           |
| `arm` GOARM=7  | `armhf`             | ARMv7 hard-float (Raspberry Pi)  |
| `386`          | `i386`              | x86 32-bit                       |

### Producing Architecture-Specific .deb Files

Each architecture gets its own `.deb` with the matching `Architecture:` field in the control file. With the manual approach, build twice:

```bash
# amd64
GOARCH=amd64 go build -o deb-amd64/usr/bin/goloo ./cmd/goloo/
# (write control with Architecture: amd64)
dpkg-deb --build deb-amd64 goloo_1.0.0-1_amd64.deb

# arm64
GOARCH=arm64 go build -o deb-arm64/usr/bin/goloo ./cmd/goloo/
# (write control with Architecture: arm64)
dpkg-deb --build deb-arm64 goloo_1.0.0-1_arm64.deb
```

With `dpkg-buildpackage`, use the `--host-arch` flag or a clean-room build:

```bash
dpkg-buildpackage --host-arch=arm64 -us -uc
sbuild -d unstable --host=arm64 --build=amd64
```

GoReleaser handles this automatically when you list multiple `goarch` values.

## Hosting an APT Repository

### Option 1: Self-Hosted with reprepro

```bash
sudo apt install reprepro
mkdir -p /var/www/apt-repo/conf
```

**conf/distributions:**

```
Origin: Goloo
Label: goloo
Codename: stable
Architectures: amd64 arm64
Components: main
Description: Goloo APT Repository
SignWith: YOUR_GPG_KEY_ID
```

**Add packages:**

```bash
reprepro -b /var/www/apt-repo includedeb stable goloo_1.0.0-1_amd64.deb
reprepro -b /var/www/apt-repo includedeb stable goloo_1.0.0-1_arm64.deb
```

This creates the standard `dists/` and `pool/` structure:

```
/var/www/apt-repo/
├── dists/
│   └── stable/
│       ├── InRelease
│       ├── Release
│       ├── Release.gpg
│       └── main/
│           ├── binary-amd64/Packages.gz
│           └── binary-arm64/Packages.gz
└── pool/
    └── main/g/goloo/
        ├── goloo_1.0.0-1_amd64.deb
        └── goloo_1.0.0-1_arm64.deb
```

Serve with nginx or Apache. Users add the repo:

```bash
curl -fsSL https://your-server.com/apt-repo/key.gpg \
  | sudo tee /etc/apt/keyrings/goloo.gpg > /dev/null

echo "deb [signed-by=/etc/apt/keyrings/goloo.gpg] https://your-server.com/apt-repo stable main" \
  | sudo tee /etc/apt/sources.list.d/goloo.list

sudo apt update && sudo apt install goloo
```

### Option 2: Self-Hosted with aptly

More feature-rich than reprepro — supports snapshots, mirrors, REST API, and publishing to S3/GCS.

```bash
aptly repo create -distribution=stable -component=main goloo-repo
aptly repo add goloo-repo goloo_1.0.0-1_amd64.deb goloo_1.0.0-1_arm64.deb
aptly snapshot create goloo-1.0.0 from repo goloo-repo
aptly publish snapshot -gpg-key=YOUR_KEY_ID goloo-1.0.0
```

### Option 3: PPA on Launchpad (Ubuntu Only)

PPAs build from source — you upload a source package and Launchpad builds for each architecture.

```bash
# Set target to Ubuntu series in debian/changelog
dch -D noble "Release for Ubuntu Noble"

# Build source-only changes
dpkg-buildpackage -S -sa -k<GPG_KEY_ID>

# Upload
dput ppa:youruser/goloo ../goloo_1.0.0-1_source.changes
```

Users add the PPA:

```bash
sudo add-apt-repository ppa:youruser/goloo
sudo apt install goloo
```

Limitations: Ubuntu only, must build from source on Launchpad's build machines, all Go dependencies must be available in the Ubuntu archive or PPA.

### Option 4: GitHub Pages

Host a static APT repo on GitHub Pages (free). Build `.deb` files in GitHub Actions, use `reprepro` to generate repo metadata, commit to a `gh-pages` branch. Users add it like any HTTP repo.

### Option 5: SaaS Services

- **Packagecloud** (packagecloud.io): `package_cloud push user/repo/ubuntu/noble goloo_1.0.0-1_amd64.deb`
- **Cloudsmith** (cloudsmith.io): REST API, supports many formats, free for open-source
- **Gemfury** (gemfury.com): `fury push goloo_1.0.0-1_amd64.deb`

All handle GPG signing and repository metadata automatically.

## GPG Signing

APT verifies packages through a chain of trust:

1. The `Release` file lists SHA256 checksums of all `Packages` index files
2. `Release` is signed, producing `Release.gpg` (detached) and `InRelease` (clearsigned, preferred by modern apt)
3. Users import the repo's public GPG key
4. `apt update` verifies the signature; `apt install` verifies package checksums against the signed index

Individual `.deb` files are not signed. Security comes from the signed repository metadata.

### Create a Signing Key

```bash
gpg --full-generate-key
# RSA 4096 bits, no expiration (or long expiration for repos)

# Export for users (binary format for /etc/apt/keyrings/)
gpg --export packages@example.com > goloo-archive-keyring.gpg
```

### Key Distribution

Modern apt (Debian 12+ / Ubuntu 22.04+) uses `signed-by` to scope keys per repository. `apt-key` is deprecated.

```bash
# User downloads and installs the key
curl -fsSL https://example.com/goloo-archive-keyring.gpg \
  | sudo tee /etc/apt/keyrings/goloo.gpg > /dev/null

# Reference in sources list
echo "deb [signed-by=/etc/apt/keyrings/goloo.gpg] https://example.com/repo stable main" \
  | sudo tee /etc/apt/sources.list.d/goloo.list
```

### Signing in CI

```bash
# Import private key from CI secret
echo "$GPG_PRIVATE_KEY" | base64 -d | gpg --batch --import

# Configure non-interactive use
echo "allow-loopback-pinentry" >> ~/.gnupg/gpg-agent.conf
```

## Getting Into Official Debian/Ubuntu

This is a significant process, typically only worthwhile for widely-adopted projects.

### The ITP Process

1. **File an ITP** (Intent to Package) bug against `wnpp` on the Debian Bug Tracking System:
   ```
   reportbug wnpp
   # Select: ITP
   ```

2. **Package all Go dependencies individually**: Debian policy requires that each Go dependency has its own `-dev` package. The AWS SDK alone has many transitive dependencies. This is the biggest hurdle.

3. **Comply with Debian Policy**: The multi-hundred-page [Debian Policy Manual](https://www.debian.org/doc/debian-policy/) plus the [Go packaging team guidelines](https://go-team.pages.debian.net/packaging.html).

4. **Find a sponsor**: New maintainers cannot upload directly. Upload to `mentors.debian.net` and request sponsorship on `debian-mentors@lists.debian.org`.

5. **NEW queue review**: First uploads are reviewed by ftpmasters for license and policy compliance. This takes weeks to months.

6. **Ongoing maintenance**: Fix bugs, handle security issues, update for new upstream versions.

Packages in Debian flow into Ubuntu's `universe` automatically. Getting into Ubuntu `main` (supported) requires a separate MIR (Main Inclusion Request).

**Practical advice**: For most projects, a self-hosted repo or PPA is the right choice. Pursue official archives only if the project has broad adoption and you're willing to maintain the packaging long-term.

## Recommended Approach for Goloo

1. **Build .deb files**: Use GoReleaser with nfpm. It handles cross-compilation and multi-arch packaging with zero Debian tooling dependencies.

2. **Host the repository**: Start with GitHub Releases (users download `.deb` directly) or a SaaS service like Cloudsmith. Graduate to self-hosted reprepro/aptly if needed.

3. **Sign the repo**: Generate a dedicated GPG key, automate signing in CI, distribute the public key with install instructions.

4. **CI automation** with GitHub Actions:

```yaml
name: Release
on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

This produces signed `.deb` files for amd64 and arm64 on every tagged release.
