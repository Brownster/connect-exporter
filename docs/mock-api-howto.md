# Mock API How-To

The mock API pretends to be AWS Connect so you can test without real AWS costs.

## Starting the server

```bash
cd docs/tools/aws-mock-api-main
npm install         # only once
node server.js
```

The server listens on `http://localhost:3000`.

## Changing its behaviour

- **Error rate** – Set `MOCK_ERROR_RATE` before starting:
  ```bash
  MOCK_ERROR_RATE=0.1 node server.js
  ```
  `0.1` means 10% of requests will randomly fail.

- **Port** – Set `PORT` if you don't want to use 3000:
  ```bash
  PORT=4000 node server.js
  ```

- **Metrics list** – Edit `metrics-config.json` in the same folder. Each entry describes how to generate a metric. Restart the server after making changes.

The API also exposes `/health` for a quick check and `/metrics` to see example output.
