# Testing

## Testing of parquet-factory

Currently the `parquet-factory` has a set of unit tests for each package. These tests are written using the [`testing`](https://golang.org/pkg/testing/) package from the Golang standard library.

For running the tests, you can use:

- `make unit_tests`, that only runs the unit tests
- `make tests`, that runs the unit and integration tests (currently only unit tests available)
- `make cover`, that will run the tests and check if the coverage is above the configured threshold
- `make before_commit`, that will run all the tests, lintian checkers and coverage checker.

### Usage of mocks

In addition, for some unit tests some mocks are needed, so the package [`gomock`](https://pkg.go.dev/github.com/golang/mock/gomock) is being used for this purpose. In order to create the test doubles needed by `gomock`, the `mockgen` program is needed.

`mockgen` is a program very related to `gomock` package, but shipped separately, that allow to generate test doubles that matches with the interfaces found in a fiven file. At the moment, it is only used to mock the whole `s3writer` package to allow testing the rest of the packages that use it without the need of manually mocking its internals.

Luckily, everything is done behind the scenes thanks to the **GNU Make** magic, so you only need to run the usual `make test`, `make unit-test`, `make cover` and the mocked interface will be generated and be ready to use by the tests.

### Integration tests (BDD)

BDD tests live inside this repository under `tests/bdd/` and are executed by the
[RedHat BDD Framework](https://github.com/LukenLarra/RedHat-BDD-Framework).

```
tests/bdd/
├── features/
│   ├── environment.py          # Behave hooks (before_all, before_scenario, …)
│   └── parquet_factory.feature # BDD scenarios for parquet-factory
└── steps/
    └── parquet_factory_steps.py # Step definitions
```

The test configuration is declared in [`framework.yml`](../framework.yml) at the root of the
repository. The CI workflow (`.github/workflows/bdd.yaml`) is fully self-contained: it
declares the required infrastructure services (Kafka, MinIO, Pushgateway), builds the
binary, and then calls the framework action.

To run the BDD tests locally:

```bash
# 1. Start infrastructure services
docker-compose up -d

# 2. Build the binary
make build

# 3. Install Python test dependencies
pip install behave confluent-kafka boto3 requests

# 4. Run the tests
make integration_tests
```

### Local testing

In order to test a local execution of the `parquet-factory`, a running Kafka and Minio instances are needed.
The provided `docker-compose.yml` will provide a quick solution to prepare the environment with the needed services,
topics and S3 buckets to be used.

The provided `config.toml` in the root directory if this repository is already configured to use the services
provided by the `docker-compose.yml`, so you just need to run:

```
docker-compose up -d
./parquet-factory
```

In order to successfully testing the program, some input data will be needed. Because the nature of the execution
of the parquet-factory, there are a few things to be considered:

- The parquet files will be only generated for Kafka messages sent in the previous hour. So, if you
  send the Kafka messages and run `parquet-factory` immediately, there are high chances that you won't see any
  generated parquet file.
- Parquet-factory will stop in only 3 conditions:
    - A message for the current hour is received in all the consumed partitions.
    - The `max_consumed_records` limit is reached (by default, it's 5000).
    - A number of `consumer_timeout` seconds without receiving any message.

In order to help injecting some valid data, the tool `tool/load_kafka.py` can be used.

#### Preparing input data

The utility `load_kafka.py` uses an especific format to read the messages to inject into Kafka topics:

```
TIMESTAMP OFFSET PARTITION PAYLOAD
```

In order to generate a number of message, `kcat` can be used:

```
kcat -b ${BOOTSTRAP_SERVERS} ${AUTHENTICATION_CONFIG} -C -t ${TOPIC} -f '%T %o %p %s\n' -o -100 -c 50 > output.txt
```

And using `load_kafka.py` to inject in our Kafka instance:

```
python tools/load_kafka.py --ignore-partition --update_timestamp output.txt kafka:9092 ${LOCAL_TOPIC}
```
