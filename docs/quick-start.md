# Quick Start

This guide explains how to run the exporter with the mock API in the simplest way possible. Pretend you are helping a friend who has never used Go or Node before.

1. **Install the requirements**
   - Install Go 1.21 from <https://go.dev/dl/>
   - Install Node.js from <https://nodejs.org>

2. **Build the exporter**
   ```bash
   go build -o connect-exporter ./cmd/connect-exporter_exporter
   ```

3. **Start the mock API**
   ```bash
   cd docs/tools/aws-mock-api-main
   npm install           # first time only
   node server.js
   ```
   You should see `Listening on http://localhost:3000`.

4. **Create a simple config**
   Save the following as `config.simple.yml` in the repository root:
   ```yaml
   listen_address: 127.0.0.1
   listen_port: 9090
   aws:
     region: "eu-west-2"
     instance_arn: "arn:aws:connect:example"
     endpoint: "http://localhost:3000"
   collectors:
     generic_realtime:
       enabled: true
       interval: 30
       collectors:
         - name: "queue_realtime_metrics"
           metrics:
             - aws_name: "AGENTS_AVAILABLE"
               prometheus_name: "connect_queue_agents"
               labels:
                 state: "Available"
           groupings: ["QUEUE", "CHANNEL"]
           filters: []
   ```

5. **Run the exporter**
   ```bash
   ENV=development ./connect-exporter --config.file=config.simple.yml --web.listen-port=9090 --labels.fqdn=localhost --labels.ip-address=127.0.0.1 --labels.location=test --labels.country=US
   ```
   Now visit `http://localhost:9090/metrics` in your browser. You should see numbers changing.

That's it! Stop either process with `Ctrl+C`.
