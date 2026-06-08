# s3-compat

Most S3 compatibility test suites are tightly coupled to a single server or buried inside a server's own CI. That makes it hard to answer the question: does this server handle the same operations the same way as the others? This repo answers that question directly.

s3-compat is a structured Go test suite that runs the same test scenarios against MinIO, SeaweedFS, Garage, RustFS, and liteio. Each server has a feature flag profile so tests skip gracefully for capabilities the server does not support. Results are collected as JSON and rendered into a markdown compatibility matrix.

## Quick start

```bash
# Copy the example config
cp configs/targets.example.yml configs/targets.yml

# Start MinIO (the default target)
docker compose -f docker/docker-compose.yml up -d minio

# Run tests against MinIO
make test

# Or run everything and generate the matrix
make test-all matrix
```

## Targets

| Name      | Version                        | S3 API port | Notes                                  |
| --------- | ------------------------------ | ----------- | -------------------------------------- |
| minio     | RELEASE.2025-07-23T15-54-02Z   | 9000        | Full-featured, path-style              |
| seaweedfs | 4.32                           | 8333        | Filer-backed S3 gateway                |
| garage    | v2.3.0                         | 3900        | Distributed, region must be "garage"   |
| rustfs    | 1.0.0-beta.7                   | 9100        | MinIO-compatible, beta feature set     |
| liteio    | local build                    | 9200        | This project's own S3 implementation   |

## Running tests

**Against a single target:**

```bash
S3COMPAT_TARGET=minio go test ./... -v -timeout 300s
```

**Against all targets (saves JSON results):**

```bash
make test-all
```

**Against real AWS S3:**

```bash
# Set credentials and point at the aws target
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
S3COMPAT_TARGET=aws go test ./... -v -timeout 300s
```

**Generate the compatibility matrix:**

```bash
make matrix
# writes COMPAT.md and prints it to stdout
```

## Adding a target

1. Add a service to `docker/docker-compose.yml`.
2. Add a profile to `configs/targets.yml` with the endpoint, credentials, and feature flags.
3. Run `S3COMPAT_TARGET=<name> go test ./... -v`.

## Test packages

| Package                     | Tests                                 | Feature flag        |
| --------------------------- | ------------------------------------- | ------------------- |
| compat/config               | Config loading, env overrides         | (always runs)       |
| compat/s3client             | Client construction                   | (always runs)       |
| scripts/gen-matrix          | Matrix generation CLI                 | (always runs)       |
| scripts/clean-buckets       | Bucket cleanup CLI                    | (always runs)       |

Feature-gated test packages (to be added) sit under `tests/<feature>/` and call `skip.Feature` at the top of each test.

## CI

GitHub Actions runs all five targets in parallel on every push to main and every pull request. Garage runs in a separate job because it requires a dynamic credential injection step. The matrix job collects all result artifacts and uploads `COMPAT.md`.

See [.github/workflows/compat.yml](.github/workflows/compat.yml) for the full pipeline.

## License

Apache-2.0
