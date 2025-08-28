# AWS Connect Exporter - Binaries

This directory contains the compiled binaries for the AWS Connect Exporter.

## Files

- **`aws-connect-exporter`** - Main production binary (compiled from `./cmd/connect-exporter_exporter`)
- **`connect-exporter`** - Alternate/legacy binary name

## Building from Source

To build the latest binary:

```bash
# From project root
go build -o bin/aws-connect-exporter ./cmd/connect-exporter_exporter

# Or use the build script
./scripts/build.sh
```

## Running the Exporter

```bash
# Basic usage
./bin/aws-connect-exporter \
  --web.listen-port=9090 \
  --labels.fqdn=your-hostname \
  --labels.location=your-location \
  --labels.country=US \
  --labels.ip-address=127.0.0.1 \
  --config.file=config.aws-connect-validated.yml \
  --no-advertise

# With development mode (uses mock API)
ENV=development ./bin/aws-connect-exporter \
  --web.listen-port=9090 \
  --labels.fqdn=localhost \
  --labels.location=test \
  --labels.country=UK \
  --labels.ip-address=127.0.0.1 \
  --config.file=configs/config.development.yml \
  --no-advertise \
  --log.level=debug
```

## Requirements

- **AWS Connect Instance**: Valid AWS Connect instance ARN
- **IAM Permissions**: See `docs/aws-connect-setup.md` for required permissions
- **Configuration**: Valid YAML configuration file

## Support

For detailed setup and configuration instructions, see:
- `docs/aws-connect-setup.md` - Complete setup guide
- `docs/quick-start.md` - Quick start guide
- `configs/README.md` - Configuration file documentation