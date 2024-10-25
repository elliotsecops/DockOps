# DockOps: Docker Image Update Checker and Minimalist CLI

DockOps is a suite of Go tools to simplify Docker management:

1. **`dockupdater`:** Checks local Docker images for updates against remote registries.
2. **`dockops`:** A minimalist CLI for common Docker operations using the Docker Engine (Moby) API.

## Table of Contents  <!-- Optional, but good practice for longer READMEs -->

- [Dockupdater](#dockupdater)
  - [Features](#dockupdater-features)
  - [Installation](#dockupdater-installation)
  - [Configuration](#dockupdater-configuration)
  - [Usage](#dockupdater-usage)
  - [Example Output](#dockupdater-example-output)
- [Dockops](#dockops)
  - [Features](#dockops-features)
  - [Installation](#dockops-installation)
  - [Usage](#dockops-usage)
- [Contributing](#contributing)
- [License](#license)

## Dockupdater: Image Update Checker

`dockupdater` efficiently checks your local Docker images for updates by comparing their tags against the latest available in remote registries (Docker Hub, GCR, etc.). It features configurable authentication, rate limiting, and retry mechanisms for robustness.

### Features

<!-- List in alphabetical order for easier scanning -->
- **Configurable Authentication:** Environment variables or `config.yaml`.
- **Concurrency:** Fast processing of multiple images.
- **JSON Output:** Easy parsing for scripts and tools. *(Planned)*
- **Local Cache:** Reduces API requests.
- **Multi-Registry Support:** Docker Hub, GCR, and more.
- **Rate Limiting:** Avoids overloading registries.
- **Retry Mechanism:** Handles transient network errors.
- **Robust Error Handling:** Informative error messages.
- **Semantic Versioning:** Accurate version comparison using SemVer.

### Installation

1. **Go:** Ensure Go is installed.
2. **Clone:** `git clone https://github.com/elliotsecops/DockOps`
3. **Navigate:** `cd DockOps`
4. **Dependencies:** `go mod tidy`
5. **Build:** `go build -o dockupdater ./cmd/updater`

### Configuration

Create `config.yaml` in the project root:

```yaml
gcr_access_token: "YOUR_GCR_ACCESS_TOKEN"
# ... other options (rate_limit, max_retries, retry_delay, output_format)
```

### Usage

```bash
./dockupdater
```

### Example Output

```
Update available for elliotsecops/my-image: 1.0.0 -> 1.0.1
// ...
```

## Dockops: Minimalist Docker CLI

`dockops` provides a streamlined CLI for essential Docker operations.

### Features

- **`list`:** List Docker images.
- **`logs`:** View container logs (`-f` to follow).
- **`remove`:** Remove Docker images.
- **`start`:** Start containers with port mapping (`-p`), volumes (`-v`), and custom commands (`-c`).
- **`stop`:** Stop running containers.

### Installation

1. **Go:** Ensure Go is installed.
2. **Clone:** `git clone https://github.com/elliotsecops/DockOps`
3. **Navigate:** `cd DockOps`
4. **Dependencies:** `go mod tidy`
5. **Build:** `go build -o dockops ./cmd/dockops`

### Usage

```bash
./dockops start <image_name> [-p <host_port>:<container_port>] [-v <host_path>:<container_path>] [-c "<command>"]
./dockops stop <container_id>
./dockops logs <container_id> [-f]  # Follow logs
./dockops remove <image_id>
./dockops list
```

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT License - see [LICENSE](LICENSE).
