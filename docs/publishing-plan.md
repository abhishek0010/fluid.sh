# Publishing Plan for Virsh Sandbox

This document outlines the strategy to package and publish `virsh-sandbox` for major Linux distributions (Debian/Ubuntu, RHEL/Fedora) and Snap.

## 1. Challenge: CGO and Libvirt

The project uses `libvirt.org/go/libvirt`, which links against the C `libvirt` library. This means:
1.  **CGO is Required:** `CGO_ENABLED=1` must be set during the build.
2.  **Dependencies:** The build environment must have `libvirt-dev` (Debian/Ubuntu) or `libvirt-devel` (RHEL) headers installed.
3.  **Cross-Compilation is Hard:** Building for `linux/arm64` on a `linux/amd64` machine requires a C cross-compiler (`gcc-aarch64-linux-gnu`) and the target architecture's libvirt libraries.

## 2. Solution: GoReleaser with Docker

We will use **GoReleaser**, the standard release automation tool for Go. To handle the CGO/Cross-compilation complexity, we will utilize GoReleaser's Docker-based build feature or a custom build image in CI.

### Recommended Toolchain
*   **GoReleaser:** Automates building binaries, creating packages (deb/rpm/snap), and publishing releases.
*   **NFPM:** (Included in GoReleaser) Handles creating `.deb` and `.rpm` packages without needing `dpkg` or `rpmbuild` present.
*   **Snapcraft:** For creating Snap packages.

## 3. Configuration Steps

### A. Create `goreleaser.yaml`
This file will be placed in the project root. It defines:
-   **Builds:** How to compile the binary (Env vars, flags, targets).
-   **Archives:** How to zip the binary for GitHub Releases (tar.gz).
-   **NFPM:** Configuration for `.deb` and `.rpm` metadata (maintainer, description, dependencies).
-   **Snap:** Configuration for Snap packages.

### B. Build Environment (GitHub Actions)
Since we need `libvirt-dev`, the GitHub Actions workflow will need to install these dependencies before running GoReleaser.
*   For **native builds** (amd64 on amd64), we simply `sudo apt-get install libvirt-dev`.
*   For **cross-builds** (arm64), we can use `zig` as a C compiler (GoReleaser supports this) OR use a Docker container with cross-compilers pre-installed. Given the library dependency, **Zig** is often the easiest modern solution if it supports the specific C headers, otherwise a Docker build strategy is safer.

## 4. Hosting Repositories

Building the `.deb` and `.rpm` files is only half the battle. Users expect to run `apt-get install` or `yum install`. This requires a **Package Repository**.

### Option 1: Cloudsmith (Recommended)
*   **Pros:** Fully managed, free for Open Source, supports Apt, Yum, Maven, Docker, etc. in one place.
*   **Setup:** Create an account, get an API key, and configure GoReleaser to push artifacts directly to Cloudsmith.
*   **User Exp:** `curl -1sLf 'https://dl.cloudsmith.io/.../setup.deb.sh' | sudo bash`

### Option 2: Gemfury
*   **Pros:** Simple, supports Apt/Yum.
*   **Cons:** Free tier has limits.

### Option 3: GitHub Releases (Manual)
*   **Pros:** Free, built-in.
*   **Cons:** Users must manually download `.deb`/`.rpm` and install with `dpkg -i` / `rpm -i`. No automatic updates via `apt upgrade`.

## 5. Implementation Roadmap

1.  **Install GoReleaser** locally to test config.
2.  **Create `goreleaser.yaml`** (I will draft this for you).
3.  **Update GitHub Action** (`.github/workflows/release.yml`) to trigger on tag creation.
4.  **Verify Cross-Compilation:** Check if we can build for ARM64 using standard runners + libraries, or if we need to restrict to AMD64 for the first iteration.

### Prerequisite Checks
Before automating, we should manually verify we can build the binary with `CGO_ENABLED=1`.

```bash
# Verify local build works
go build -v -o virsh-sandbox-api ./cmd/api
```
