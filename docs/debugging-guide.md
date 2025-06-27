# Debugging and Logs

Both the exporter and the mock API print logs to the terminal. You can save them to a file with `>`.

## Exporter

Run with debug logging:

```bash
./connect-exporter --config.file=config.yml --log.level=debug > exporter.log 2>&1
```

- `--log.level=debug` prints extra details.
- The logs above are saved to `exporter.log`.

To watch the log in real time run:

```bash
tail -f exporter.log
```

## Mock API

Start the mock API with verbose logs:

```bash
DEBUG=* node server.js > mock-api.log 2>&1
```

Check the log file with `tail -f mock-api.log`.

When something goes wrong, these logs are the first place to look.
