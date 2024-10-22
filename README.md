# Dockops, a Docker Image Update Checker

This Go script checks for updates to your local Docker images by comparing their tags against the latest tags available in remote registries. It supports Docker Hub, Google Container Registry (GCR), and other registries, using a configurable authentication mechanism and robust error handling.

## Features

* **Supports Multiple Registries:** Checks for updates on Docker Hub, GCR, and other registries (you can add more!).
* **Semantic Versioning:** Uses semantic versioning (SemVer) for accurate version comparison.
* **Configurable Authentication:** Supports authentication using environment variables or a configuration file (YAML).
* **Rate Limiting:** Prevents overloading remote registries by limiting the rate of requests.
* **Retry Mechanism:** Includes retry logic with exponential backoff for transient network errors.
* **JSON Output:** Provides output in JSON format for easy parsing by other scripts or tools.
* **Robust Error Handling:** Provides informative error messages for improved debugging.
* **Concurrency:** Processes images concurrently to improve performance.
* **Local Cache:** Caches remote tag information to reduce the number of API requests.


## Installation

1. Ensure you have Go installed.
2. Clone the repository: `git clone https://github.com/elliotsecops/DockOps`
3. Navigate to the project directory: `cd docker-image-checker`
4. Install dependencies: `go mod tidy`
5. Build the executable: `go build -o dockops`


## Configuration

Create a `config.yaml` file in the same directory as the executable. Here's an example:

```yaml
gcr_access_token: "YOUR_GCR_ACCESS_TOKEN" # Required for GCR images. Get this from Google Cloud Console.
rate_limit: 1s                         # Rate limit (e.g., 1s, 100ms). Defaults to 100ms if not set.
max_retries: 3                         # Maximum number of retries for network errors. Defaults to 3 if not set.
retry_delay: 1s                        # Delay between retries (e.g., 1s, 200ms). Defaults to 1s if not set.
output_format: "text"                  # Output format ("text" or "json"). Defaults to "text" if not set.
```

You can also set these configuration options via environment variables (e.g., `GCR_ACCESS_TOKEN`, `RATE_LIMIT`, `MAX_RETRIES`, `RETRY_DELAY`, `OUTPUT_FORMAT`). Environment variables override the values in `config.yaml`.

## Usage

Run the executable: `./dockops`

The script will output the update status for each image, indicating whether it's up-to-date or requires an update.

**Example Output (Text Format):**  (This example shows images from your Docker Hub account)

```
Image elliotsecops/my-image:latest is up-to-date.
Image elliotsecops/another-image:v1.0.0 is outdated. Latest version is v1.0.1.
Image library/ubuntu:latest is outdated. Latest version is 22.04.
Error checking gcr.io/my-project/my-image:v1.2.3: error getting remote tags: ...
```

**Example Output (JSON Format):**

```json
{"update":"Image elliotsecops/my-image:latest is up-to-date."}
{"update":"Image elliotsecops/another-image:v1.0.0 is outdated. Latest version is v1.0.1."}
{"update":"Error checking gcr.io/my-project/my-image:v1.2.3: error getting remote tags: ..."}
```

## Contributing

Contributions are welcome! Please open issues or submit pull requests.

## License

MIT
