# BDD Framework Setup — parquet-factory

This document describes all the changes made in this repository to migrate from
the centralised BDD workflow (delegated to `RedHatInsights/processing-tools`) to
a self-contained workflow that runs the
[RedHat-BDD-Framework](https://github.com/LukenLarra/RedHat-BDD-Framework)
directly inside the `parquet-factory` repository.

---

## Table of Contents

1. [Context — what changed and why](#1-context--what-changed-and-why)
2. [Repository structure affected](#2-repository-structure-affected)
3. [framework.yml](#3-frameworkyml)
4. [.github/workflows/bdd.yaml](#4-githubworkflowsbddyaml)
    - [Services](#services)
    - [Global environment variables](#global-environment-variables)
    - [Steps](#steps)
5. [Path management](#5-path-management)
6. [Known quirks and important notes](#6-known-quirks-and-important-notes)

---

## 1. Context — what changed and why

**Before (main branch):**

```yaml
jobs:
    bdd:
        uses: RedHatInsights/processing-tools/.github/workflows/bdd.yaml@master
        with:
            service: parquet-factory
            use_kafka: true
            use_minio: true
            use_pushgateway: true
```

The workflow was fully delegated to the `processing-tools` centralised reusable
workflow. This made it impossible to customise infrastructure or test execution
without modifying that shared repository.

**After (this branch):**

All infrastructure services (Kafka, MinIO, Pushgateway) are declared and started
directly in `.github/workflows/bdd.yaml`. Test execution is driven by
`LukenLarra/RedHat-BDD-Framework` via the `framework.yml` configuration file at
the root of this repository.

---

## 2. Repository structure affected

```
.github/
  workflows/
    bdd.yaml                      ← completely rewritten
framework.yml                     ← new file (BDD Framework configuration)
tests/
  bdd/
    features/
      environment.py              ← new file (forwards hooks to insights-behavioral-spec)
      steps/
        parquet_factory.py        ← updated paths and binary resolution
```

The shared step definitions and `environment.py` hooks live in
[LukenLarra/insights-behavioral-spec](https://github.com/LukenLarra/insights-behavioral-spec)
on the branch `descentralize-bdd-test-execution`. They are checked out at runtime
into the `insights-behavioral-spec/` directory at the repository root.

---

## 3. framework.yml

Located at the **repository root**. Configures what the BDD Framework action
runs when invoked.

```yaml
project:
    name: "parquet-factory"
    version: "1.0.0"

tests:
    enabled: true
    path: "."
    command: "python -m behave tests/bdd/features --junit --junit-directory reports/junit --format pretty"
    bdd:
        features: "tests/bdd/features"
        steps: "tests/bdd/features/steps"
        environment: "tests/bdd/features/environment.py"
    env:
        # — see full table in §4 —
```

Key points:

| Field                   | Value                                   | Notes                                                   |
| ----------------------- | --------------------------------------- | ------------------------------------------------------- |
| `tests.path`            | `.`                                     | The framework runs behave from the repository root.     |
| `tests.command`         | `python -m behave tests/bdd/features …` | Path is relative to the repository root.                |
| `tests.bdd.features`    | `tests/bdd/features`                    | Where `.feature` files live.                            |
| `tests.bdd.steps`       | `tests/bdd/features/steps`              | Project-specific step definitions.                      |
| `tests.bdd.environment` | `tests/bdd/features/environment.py`     | Forwards hooks to the shared `environment.py` (see §5). |

The `tests.env` block mirrors the job-level `env:` in the workflow so that the
framework process inherits the same variables even if it runs in a sub-process.

---

## 4. .github/workflows/bdd.yaml

### Trigger

```yaml
on:
    push:
        branches: ["main"]
    pull_request:
    workflow_dispatch: # allows manual execution via gh CLI or GitHub UI
```

> **Note:** `workflow_dispatch` is required to run the workflow manually with
> `gh workflow run bdd.yaml …`. Without it the CLI returns
> _"could not find any workflows"_.

### Permissions

```yaml
permissions:
    contents: read
    checks: write
    pull-requests: write
```

`checks: write` and `pull-requests: write` are needed so the BDD Framework
action can publish JUnit results as check annotations on the PR.

---

### Services

Only **Kafka** is declared as a GitHub Actions service container. MinIO and
Pushgateway are started manually in dedicated steps (see below) because GitHub
Actions service containers do not support the `command` or `args` keys — those
are Kubernetes concepts. Passing custom arguments to a Docker entrypoint requires
running the container with `docker run` in a step.

#### Kafka service container

```yaml
services:
    kafka:
        image: confluentinc/cp-kafka
        ports:
            - 9092:9092
        env:
            KAFKA_NODE_ID: 1
            KAFKA_PROCESS_ROLES: broker,controller
            KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
            KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
            KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT
            KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
            KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
            KAFKA_CONTROLLER_QUORUM_VOTERS: 1@localhost:9093
            KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
            KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
            KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
            KAFKA_LOG4J_ROOT_LOGLEVEL: WARN
            KAFKA_TOOLS_LOG4J_LOGLEVEL: ERROR
            CLUSTER_ID: test_id
        options: >-
            --health-cmd "kafka-broker-api-versions --bootstrap-server localhost:9092"
            --health-interval 10s
            --health-timeout 10s
            --health-retries 10
```

The runner waits for the health check to pass before executing any step.

---

### Global environment variables

These are declared at **job level** under `env:` and are available to every step
and to the BDD Framework sub-process.

#### insights-behavioral-spec / behave

| Variable     | Value                                                                                                                                      | Purpose                                                                                                                                                                                                             |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PYTHONPATH` | `insights-behavioral-spec/features/src:insights-behavioral-spec/features/steps:insights-behavioral-spec/features:insights-behavioral-spec` | Makes shared step definitions, `src/` helpers and the shared `environment.py` importable by both behave and any Python subprocess. All paths are relative to the repository root (the working directory during CI). |

#### Kafka (consumed by `insights-behavioral-spec/features/environment.py`)

| Variable     | Value       | Purpose                |
| ------------ | ----------- | ---------------------- |
| `KAFKA_HOST` | `localhost` | Kafka broker hostname. |
| `KAFKA_PORT` | `9092`      | Kafka broker port.     |

#### S3 / MinIO (consumed by `insights-behavioral-spec/features/environment.py`)

| Variable                       | Value                    | Purpose                                                         |
| ------------------------------ | ------------------------ | --------------------------------------------------------------- |
| `S3_TYPE`                      | `minio`                  | Tells the shared steps to use MinIO-compatible S3.              |
| `S3_HOST`                      | `localhost`              | MinIO host.                                                     |
| `S3_PORT`                      | `9000`                   | MinIO API port.                                                 |
| `S3_ACCESS_KEY_ID`             | `test_access_key`        | MinIO root user.                                                |
| `S3_SECRET_ACCESS_KEY`         | `test_secret_access_key` | MinIO root password.                                            |
| `S3_BUCKET`                    | `test`                   | Bucket used by both MinIO and `parquet-factory`.                |
| `S3_OLDER_MINIO_COMPATIBILITY` | `1`                      | Enables path-style addressing required by older MinIO versions. |

#### parquet-factory binary configuration

These variables are read directly by the `parquet-factory` binary as
configuration overrides. They are also inherited by any subprocess spawned from
the step definitions.

| Variable                                | Value                    | Purpose                                                                                                                                                              |
| --------------------------------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PARQUET_FACTORY__KAFKA_RULES__ADDRESS` | `localhost:9092`         | Kafka broker address for the rules topic consumer.                                                                                                                   |
| `PARQUET_FACTORY__S3__ENDPOINT`         | `localhost:9000`         | S3-compatible (MinIO) endpoint.                                                                                                                                      |
| `PARQUET_FACTORY__S3__BUCKET`           | `test`                   | Target S3 bucket.                                                                                                                                                    |
| `PARQUET_FACTORY__S3__ACCESS_KEY`       | `test_access_key`        | S3 access key.                                                                                                                                                       |
| `PARQUET_FACTORY__S3__SECRET_KEY`       | `test_secret_access_key` | S3 secret key.                                                                                                                                                       |
| `PARQUET_FACTORY__S3__USE_SSL`          | `false`                  | Disables TLS for local MinIO.                                                                                                                                        |
| `PARQUET_FACTORY__S3__PREFIX`           | `fleet_aggregations`     | Object key prefix inside the bucket.                                                                                                                                 |
| `PARQUET_FACTORY__S3__REGION`           | `us-east-1`              | AWS region value (required by the S3 SDK even for MinIO).                                                                                                            |
| `PARQUET_FACTORY__METRICS__GATEWAY_URL` | `localhost:9091`         | Prometheus Pushgateway address.                                                                                                                                      |
| `PARQUET_FACTORY_BIN`                   | `./parquet-factory`      | Path to the compiled binary. The step definitions read this variable via `os.environ.get("PARQUET_FACTORY_BIN", "parquet-factory")` instead of hard-coding the name. |

---

### Steps

The steps execute in the following order:

| #   | Step name                                               | What it does                                                                                                                                                                                                                                                                                                                                                                             |
| --- | ------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Checkout repository**                                 | Checks out `parquet-factory` into the runner workspace root.                                                                                                                                                                                                                                                                                                                             |
| 2   | **Checkout shared steps from insights-behavioral-spec** | Checks out `LukenLarra/insights-behavioral-spec@descentralize-bdd-test-execution` into `insights-behavioral-spec/` (relative to the workspace root). This directory is added to `PYTHONPATH`.                                                                                                                                                                                            |
| 3   | **Set up Go**                                           | Installs the Go version declared in `go.mod` with module cache enabled.                                                                                                                                                                                                                                                                                                                  |
| 4   | **Generate mocks**                                      | Runs `make gen_mocks` to produce the mock files required to compile the project.                                                                                                                                                                                                                                                                                                         |
| 5   | **Build parquet-factory binary**                        | Runs `./build.sh` and produces `./parquet-factory` in the workspace root. The binary path is referenced by `PARQUET_FACTORY_BIN`.                                                                                                                                                                                                                                                        |
| 6   | **Start MinIO**                                         | Starts `minio/minio` as a detached Docker container, exposing ports `9000` (API) and `9001` (web console). Waits up to 60 s for the health endpoint `http://localhost:9000/minio/health/live` to respond.                                                                                                                                                                                |
| 7   | **Start Pushgateway**                                   | Starts `prom/pushgateway` as a detached Docker container with `--web.enable-admin-api`. The admin API is needed so the shared steps can reset metrics between test scenarios. Waits up to 30 s for `http://localhost:9091/-/healthy`. Launched via `docker run` (not as a service container) because service containers do not support passing custom arguments to the image entrypoint. |
| 8   | **Create Kafka topics**                                 | Creates `incoming_rules_topic` (2 partitions) and `incoming_features_topic` (4 partitions) using a temporary `confluentinc/cp-kafka` container on `--network host`.                                                                                                                                                                                                                      |
| 9   | **Create MinIO bucket**                                 | Uses `minio/mc` to register the MinIO alias and create the `test` bucket.                                                                                                                                                                                                                                                                                                                |
| 10  | **Install kcat**                                        | Installs `kafkacat` (kcat) via `apt-get`. Used by the shared steps to produce Kafka messages during test scenarios.                                                                                                                                                                                                                                                                      |
| 11  | **Run BDD Framework**                                   | Invokes `LukenLarra/RedHat-BDD-Framework@main`. The action reads `framework.yml` from the repository root, runs behave, and publishes JUnit results as check annotations.                                                                                                                                                                            |

---

## 5. Path management

All paths must be evaluated relative to the **repository root**, which is the
working directory throughout the entire job.

### PYTHONPATH composition

```
insights-behavioral-spec/features/src       ← shared helper modules (e.g. process_output)
insights-behavioral-spec/features/steps     ← shared step definitions
insights-behavioral-spec/features           ← allows `import environment` from the shared package
insights-behavioral-spec                    ← top-level package root
```

These four entries are colon-separated and set both in the workflow `env:` block
and in `framework.yml tests.env` so that every process (behave, the BDD
Framework action, and any subprocess) can resolve the imports without additional
configuration.

### environment.py forwarding (`tests/bdd/features/environment.py`)

Behave looks for `environment.py` in the features directory
(`tests/bdd/features/`). The file in that location does **not** contain hooks
directly — it dynamically loads the shared `environment.py` from
`insights-behavioral-spec/features/environment.py` using a path computed
relative to `__file__`:

```python
_THIS_DIR     = os.path.dirname(os.path.abspath(__file__))
_PROJECT_ROOT = os.path.normpath(os.path.join(_THIS_DIR, "..", "..", ".."))
_SHARED_ENV   = os.path.join(_PROJECT_ROOT, "insights-behavioral-spec", "features", "environment.py")
```

This means the forwarding works correctly regardless of the current working
directory at the time behave is invoked.

### DATA_DIRECTORY in step definitions (`tests/bdd/features/steps/parquet_factory.py`)

Test data (JSON message templates, etc.) is located in `testdata/` at the
repository root. The step definitions resolve the path anchored to `__file__`:

```python
DATA_DIRECTORY = os.path.join(
    os.path.dirname(os.path.abspath(__file__)),
    "..", "..", "..", "..",   # steps/ → features/ → bdd/ → tests/ → repo root
    "testdata"
)
```

This path was corrected from `test_data` (old name) to `testdata` (actual
directory name) as part of this migration.

---

## 6. Known quirks and important notes

### Triggering the workflow manually

Because GitHub reads workflow names from the default branch (`main`), and `main`
still contains the old delegated workflow, `gh workflow run "BDD tests"` returns
_"could not find any workflows named BDD tests"_. Use the filename instead:

```bash
gh workflow run bdd.yaml --ref centralize-parquet-factory-test-execution --repo <owner>/parquet-factory
```

### service containers cannot receive custom arguments

GitHub Actions service containers only support the keys `image`, `credentials`,
`env`, `ports`, `volumes`, and `options`. The `command` and `args` keys used in
Kubernetes / `docker-compose` are **not supported**. Any image that requires
custom entrypoint arguments (such as Pushgateway with `--web.enable-admin-api`)
must be started as a manual `docker run` step instead.

### Pushgateway admin API requirement

The shared steps in `insights-behavioral-spec` call the Pushgateway admin API
(`DELETE /api/v1/admin/wipe`) to reset all metrics before each scenario that
checks Prometheus metrics. Without `--web.enable-admin-api`, those requests
return `404` and the tests fail.

### MinIO vs AWS S3

Setting `S3_OLDER_MINIO_COMPATIBILITY=1` forces path-style requests
(`http://host:port/bucket/key`) instead of virtual-hosted-style
(`http://bucket.host:port/key`). This is required when using a local MinIO
instance that is not behind a DNS-capable load balancer.
