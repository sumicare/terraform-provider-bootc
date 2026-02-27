## OpenTofu Provider Bootc

A Terraform/OpenTofu provider for building bootable disk images from [bootc](https://github.com/bootc-dev/bootc) container images.
Uses the modern Terraform Plugin Framework with an embedded Rust bridge to [bootc-lib](https://github.com/bootc-dev/bootc/tree/main/crates/lib).

## Features

- Modern Terraform Plugin Framework (not legacy SDKv2)
- Build bootable disk images from bootc container images
- Embedded Rust bridge to bootc-lib (no external bootc binary required)
- Output qcow2 disk images (via `qemu-img`)
- Configurable disk size, filesystem type, and bootloader
- Support for kernel arguments and SSH key injection

## Prerequisites

- `qemu-img` (for disk image conversion)
- Podman (for pulling container images)

## Quick Start

```hcl
terraform {
  required_providers {
    bootc = {
      source  = "sumicare/bootc"
      version = "~> 0.1.0"
    }
  }
}

provider "bootc" {}

resource "bootc_image" "example" {
  source_image = "quay.io/fedora/fedora-bootc:42"
  output_path  = "/tmp/images"
}
```

## Resource: `bootc_image`

The `bootc_image` resource builds a qcow2 disk image from a bootc container image.

### Required Arguments

| Name | Type | Description |
|------|------|-------------|
| `source_image` | string | Container image reference (e.g. `quay.io/fedora/fedora-bootc:42`) |
| `output_path` | string | Directory where the disk image will be written |

### Optional Arguments

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `disk_size` | string | `"1G"` | Total raw disk image size (supports K, M, G, T suffixes) |
| `output_filename` | string | `"disk.qcow2"` | Filename for the resulting qcow2 image |
| `filesystem` | string | - | Root filesystem type: `xfs`, `ext4`, or `btrfs` |
| `root_size` | string | - | Size of root partition (M/G/T suffixes). Default uses all remaining space |
| `kargs` | list(string) | - | Kernel arguments (e.g. `["console=ttyS0,115200n8"]`) |
| `root_ssh_authorized_keys` | string | - | Path to authorized_keys file to inject into root account |
| `target_imgref` | string | - | Container image reference for subsequent bootc upgrades |
| `disable_selinux` | bool | `false` | Disable SELinux in the installed system |
| `generic_image` | bool | `true` | Build generic image with all bootloader types, skip firmware changes |
| `bootloader` | string | - | Bootloader to use: `grub`, `systemd`, or `none` |

### Computed Attributes

| Name | Type | Description |
|------|------|-------------|
| `image_path` | string | Full path to the resulting qcow2 file |

### Example with Options

```hcl
resource "bootc_image" "server" {
  source_image     = "quay.io/fedora/fedora-bootc:42"
  output_path      = "/var/lib/images"
  output_filename  = "server.qcow2"
  disk_size        = "20G"
  filesystem       = "xfs"
  root_size        = "10G"
  bootloader       = "grub"
  generic_image    = true
  
  kargs = [
    "console=ttyS0,115200n8",
    "net.ifnames=0"
  ]
  
  root_ssh_authorized_keys = "${path.module}/authorized_keys"
}
```

### Behavior

1. Creates a sparse raw disk file using `truncate`
2. Runs `bootc install to-disk --via-loopback` with the specified options
3. Converts the raw disk to qcow2 using `qemu-img convert`
4. Removes the intermediate raw file

**Note**: The resource is immutable. Any changes require replacement (destroy and recreate).

## Development

### Prerequisites

- Go 1.25+
- Rust toolchain (stable)
- Podman

### Building

```bash
# Build the Rust bridge first
cargo build --release -p bootc-bridge

# Build the Go provider
CGO_ENABLED=1 go build -o terraform-provider-bootc
```

### Local Development with OpenTofu

For local development and testing, use the `.tofurc` configuration to override the provider with your local build:

1. **Build the provider**:
   ```bash
   cargo build --release -p bootc-bridge
   CGO_ENABLED=1 go build -o terraform-provider-bootc
   ```

2. **Set up the dev override**:
   ```bash
   export TF_CLI_CONFIG_FILE=/path/to/.tofurc
   ```

3. **Update the path in `.tofurc`** to point to your local build directory:
   ```hcl
   provider_installation {
     dev_overrides {
       "sumicare/bootc" = "/path/to/opentofu-provider-bootc"
     }
     direct {}
   }
   ```

**Note**: When using `dev_overrides`, OpenTofu will skip provider version checks and use your local binary directly.

### Testing

```bash
# Unit tests
CGO_ENABLED=1 go test -v -cover -timeout=120s ./...

# Acceptance tests (requires Podman)
CGO_ENABLED=1 TF_ACC=1 go test -v -cover -timeout=20m ./...
```

## License

Sumicare OpenTofu Provider Bootc is licensed under the terms of [Apache License 2.0](LICENSE).

The embedded `bootc-src` directory contains vendored source from [bootc](https://github.com/bootc-dev/bootc) which is licensed under both the Apache License 2.0 and the MIT License.
