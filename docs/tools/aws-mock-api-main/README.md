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
| `MOCK_ERROR_RATE`   | `0.2`                                       | Probability (0-1) that API requests fail with an error       |
| `ITEMS_PER_PAGE`    | `2`                                         | Number of items returned in paginated responses              |

Example:

```bash
MOCK_ERROR_RATE=0.5 ITEMS_PER_PAGE=5 node server.js
```

## Historical metrics

Values returned for `CONTACTS_QUEUED`, `CONTACTS_ABANDONED`, `CONTACTS_HANDLED` and `CONTACTS_HANDLED_OUTBOUND` accumulate over time. The totals reset when the server is restarted. If the `/admin/resetTotals` endpoint is enabled you can POST to it to clear the counters without restarting.
