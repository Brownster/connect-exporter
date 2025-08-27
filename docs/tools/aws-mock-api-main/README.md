# AWS Connect Mock API

This directory contains a simple Express server that emulates a subset of the AWS Connect API. It is useful for local development and testing without incurring AWS costs.

## Starting the server

```bash
cd docs/tools/aws-mock-api-main
npm install    # first run only
node server.js
```

The server listens on `http://localhost:3000` by default. Set the `PORT` environment variable to override the port.

## Configuration via environment variables

The mock server can be customised using the following variables. Each one has a sensible default if not provided:

| Variable            | Default                                    | Description                                                  |
|---------------------|--------------------------------------------|--------------------------------------------------------------|
| `MOCK_INSTANCE_ID`  | `f604177c-b2ab-4c76-9033-635d195b2772`      | Instance ID used in generated ARNs and validation            |
| `MOCK_ACCOUNT_ID`   | `123456789012`                              | Account ID portion of generated ARNs                         |
| `MOCK_REGION`       | `eu-west-2`                                 | AWS region for generated ARNs                                |
| `MOCK_ERROR_RATE`   | `0.0`                                       | Probability (0-1) that API requests fail with an error       |
| `ITEMS_PER_PAGE`    | `2`                                         | Number of items returned in paginated responses              |
| `MOCK_QUEUE_COUNT`  | `120`                                       | Number of queues to generate for batching tests              |

Example:

```bash
MOCK_ERROR_RATE=0.1 ITEMS_PER_PAGE=5 MOCK_QUEUE_COUNT=120 node server.js
```

## Testing Batching Logic

The mock API now generates **120 queues by default** to test the exporter's batching logic:

- **120 queues** → **2 API requests** (100 + 20 queues per batch)
- Each queue has realistic metadata (departments, channels, staffing targets)
- Queue types: VOICE, CHAT, TASK across 10 departments
- Simulates real-world Connect instance with many queues

### Batching Test Commands

```bash
# Start mock API with 120 queues for batching tests
MOCK_QUEUE_COUNT=120 MOCK_ERROR_RATE=0.0 node server.js

# Test with even more queues to verify batching scales
MOCK_QUEUE_COUNT=250 node server.js  # → 3 API requests (100 + 100 + 50)
```

### Real API Testing Command

To test against the real Kensington AWS Connect API (provided by user):

```bash
AWS_PROFILE=kensington AWS_REGION=eu-west-2 ./connect-exporter \
  --web.listen-port=8543 \
  --labels.fqdn=localhost \
  --labels.location=test \
  --labels.ip-address=127.0.0.1 \
  --config.file=config.aws-connect-validated-updated.yml \
  --no-advertise \
  --instance-arn=arn:aws:connect:eu-west-2:851725229028:instance/207eb2 \
  --log.level=debug
```

## Historical metrics

Values returned for `CONTACTS_QUEUED`, `CONTACTS_ABANDONED`, `CONTACTS_HANDLED` and `CONTACTS_HANDLED_OUTBOUND` accumulate over time. The totals reset when the server is restarted. If the `/admin/resetTotals` endpoint is enabled you can POST to it to clear the counters without restarting.
