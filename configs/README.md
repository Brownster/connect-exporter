# Configuration Files

This folder contains various configuration files for the AWS Connect Exporter.

## Configuration Files

### Environment-Specific Configs
- `config.development.yml` - Development environment configuration with mock API settings
- `config.test.yml` - Test environment configuration for automated testing
- `config.batching-test.yml` - Specialized config for testing queue batching with 30s intervals

### Customer-Specific Configs
- `config.kensington-production.yml` - Production configuration for Kensington environment
- `config.kensington-test.yml` - Test configuration for Kensington environment

### Base Configurations
- `config.aws-connect.yml` - Basic AWS Connect configuration template

## Active Configuration

The validated configuration file is kept in the project root for easy testing access:
- `../config.aws-connect-validated-updated.yml` - Currently validated and tested configuration

## Usage

Use any configuration file with the `--config.file` flag:

```bash
./bin/connect-exporter --config.file=configs/config.development.yml
```

## Configuration Structure

All configurations follow the same YAML structure:

```yaml
aws:
  region: "your-region"
  instance_arn: "your-instance-arn"

collectors:
  realtime_queue:
    enabled: true
    interval: 60
  agent_status:
    enabled: true
    interval: 300
  # ... other collectors
```

## Creating New Configurations

1. Copy an existing config that matches your use case
2. Update the AWS settings for your environment
3. Adjust collector intervals as needed
4. Test with your specific AWS Connect instance