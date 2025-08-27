# Built Binaries

This folder contains compiled AWS Connect Exporter binaries.

## Current Binary
- `connect-exporter` - Latest build with all enhancements (built from main)

## Previous Versions
- `connect-exporter-enhanced` - Enhanced version with agent caching and filtering
- `connect-exporter-lab-ready` - Lab deployment ready version
- `connect-exporter_100maxissue` - Version addressing 100+ queue issues
- `connect-exporter_27_08_25` - Timestamped backup version

## Building New Binaries

Use the build script to create new binaries:

```bash
../misc/build.sh
```

This will create a new `connect-exporter` binary with the latest code.