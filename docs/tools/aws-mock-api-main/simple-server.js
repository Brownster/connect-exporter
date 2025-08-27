/**
 * Simplified AWS Connect Mock API Server
 * 
 * This provides basic AWS Connect API simulation for development testing.
 * Endpoints:
 * - POST / (with x-amz-target header for AWS SDK calls)
 * - GET /health (health check)
 * - POST /admin/errorRate (configure error simulation)
 */

const http = require('http');
const url = require('url');

// Mock data
const INSTANCE_ID = 'f604177c-b2ab-4c76-9033-635d195b2772';
const PORT = 3000;

// Mock queues data
const MOCK_QUEUES = [
    {
        Id: 'queue-001',
        Name: 'Sales Queue',
        QueueType: 'STANDARD',
        InstanceId: INSTANCE_ID
    },
    {
        Id: 'queue-002', 
        Name: 'Support Queue',
        QueueType: 'STANDARD',
        InstanceId: INSTANCE_ID
    },
    {
        Id: 'queue-003',
        Name: 'Monitoring Voice Queue',
        QueueType: 'STANDARD',
        InstanceId: INSTANCE_ID
    },
    {
        Id: 'queue-004',
        Name: 'Monitoring Chat Queue', 
        QueueType: 'STANDARD',
        InstanceId: INSTANCE_ID
    }
];

// Mock agent statuses
const MOCK_AGENT_STATUSES = [
    {
        Id: 'status-001',
        Name: 'Available',
        Type: 'ROUTABLE'
    },
    {
        Id: 'status-002',
        Name: 'Busy',
        Type: 'ROUTABLE'
    },
    {
        Id: 'status-003',
        Name: 'After Contact Work',
        Type: 'ROUTABLE'
    },
    {
        Id: 'status-004',
        Name: 'Custom Break',
        Type: 'CUSTOM'
    }
];

// Error simulation
let errorRate = 0.1;

function shouldSimulateError() {
    return Math.random() < errorRate;
}

function generateMetrics(queueIds) {
    const metrics = [];
    
    queueIds.forEach(queueId => {
        const queue = MOCK_QUEUES.find(q => q.Id === queueId);
        if (queue) {
            metrics.push({
                Dimensions: {
                    Queue: {
                        Id: queueId,
                        Name: queue.Name,
                        Type: queue.QueueType
                    }
                },
                Collections: [
                    {
                        Metric: {
                            Name: 'CONTACTS_IN_QUEUE',
                            Unit: 'COUNT'
                        },
                        Value: Math.floor(Math.random() * 20) // 0-19 contacts
                    },
                    {
                        Metric: {
                            Name: 'OLDEST_CONTACT_AGE',
                            Unit: 'SECONDS'
                        },
                        Value: Math.floor(Math.random() * 600) // 0-600 seconds
                    }
                ]
            });
        }
    });
    
    return metrics;
}

const server = http.createServer((req, res) => {
    const parsedUrl = url.parse(req.url, true);
    const path = parsedUrl.pathname;
    const method = req.method;
    
    // Enable CORS
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', '*');
    
    if (method === 'OPTIONS') {
        res.writeHead(200);
        res.end();
        return;
    }
    
    console.log(`${method} ${path}`);
    console.log('Headers:', req.headers);
    
    // Health check endpoint
    if (path === '/health') {
        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.end('OK');
        return;
    }
    
    // Admin endpoint to set error rate
    if (path === '/admin/errorRate' && method === 'POST') {
        let body = '';
        req.on('data', chunk => body += chunk);
        req.on('end', () => {
            try {
                const data = JSON.parse(body);
                errorRate = data.rate || 0;
                console.log('Error rate set to:', errorRate);
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ success: true, errorRate }));
            } catch (e) {
                res.writeHead(400, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ error: 'Invalid JSON' }));
            }
        });
        return;
    }
    
    // Handle REST-style queue endpoints (GET requests)
    if (method === 'GET' && path.includes('/queues-summary/')) {
        console.log('Handling REST-style ListQueues request');
        
        if (shouldSimulateError()) {
            console.log('Simulating error response');
            res.writeHead(500, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({
                __type: 'InternalServiceException',
                message: 'Simulated server error'
            }));
            return;
        }
        
        const response = {
            QueueSummaryList: MOCK_QUEUES.map(queue => ({
                Id: queue.Id,
                Name: queue.Name,
                QueueType: queue.QueueType
            }))
        };
        
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify(response));
        return;
    }
    
    // Handle REST-style metrics endpoints (POST requests)
    if (method === 'POST' && path.includes('/metrics/current/')) {
        console.log('Handling REST-style GetCurrentMetricData request');
        
        if (shouldSimulateError()) {
            console.log('Simulating error response');
            res.writeHead(500, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({
                __type: 'InternalServiceException',
                message: 'Simulated server error'
            }));
            return;
        }
        
        let body = '';
        req.on('data', chunk => body += chunk);
        req.on('end', () => {
            try {
                const requestData = body ? JSON.parse(body) : {};
                console.log('GetCurrentMetricData body:', body);
                
                const filters = requestData.Filters;
                const queueIds = filters?.Queues || [];
                const metrics = generateMetrics(queueIds);
                
                const response = {
                    MetricResults: metrics
                };
                
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify(response));
            } catch (e) {
                console.error('Error parsing GetCurrentMetricData request:', e);
                res.writeHead(400, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({
                    __type: 'InvalidParameterException',
                    message: 'Invalid request format'
                }));
            }
        });
        return;
    }
    
    // Handle REST-style agent status endpoints (GET requests)
    if (method === 'GET' && path.includes('/agent-status/')) {
        console.log('Handling REST-style ListAgentStatuses request');
        
        if (shouldSimulateError()) {
            console.log('Simulating error response');
            res.writeHead(500, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({
                __type: 'InternalServiceException',
                message: 'Simulated server error'
            }));
            return;
        }
        
        const response = {
            AgentStatusSummaryList: MOCK_AGENT_STATUSES
        };
        
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify(response));
        return;
    }
    
    // AWS Connect API endpoints (JSON over POST)
    if (method === 'POST' && path === '/') {
        const action = req.headers['x-amz-target'];
        
        let body = '';
        req.on('data', chunk => body += chunk);
        req.on('end', () => {
            console.log('Body:', body);
            
            if (shouldSimulateError()) {
                console.log('Simulating error response');
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({
                    __type: 'InternalServiceException',
                    message: 'Simulated server error'
                }));
                return;
            }
            
            try {
                const requestData = body ? JSON.parse(body) : {};
                
                if (action === 'Connect_20170801.ListQueues') {
                    console.log('Handling ListQueues request');
                    
                    const instanceId = requestData.InstanceId;
                    if (!instanceId || !instanceId.includes(INSTANCE_ID)) {
                        res.writeHead(400, { 'Content-Type': 'application/json' });
                        res.end(JSON.stringify({
                            __type: 'InvalidParameterException',
                            message: 'Invalid instance ID'
                        }));
                        return;
                    }
                    
                    const response = {
                        QueueSummaryList: MOCK_QUEUES.map(queue => ({
                            Id: queue.Id,
                            Name: queue.Name,
                            QueueType: queue.QueueType
                        }))
                    };
                    
                    res.writeHead(200, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify(response));
                    
                } else if (action === 'Connect_20170801.GetCurrentMetricData') {
                    console.log('Handling GetCurrentMetricData request');
                    
                    const instanceId = requestData.InstanceId;
                    const filters = requestData.Filters;
                    
                    if (!instanceId || !instanceId.includes(INSTANCE_ID)) {
                        res.writeHead(400, { 'Content-Type': 'application/json' });
                        res.end(JSON.stringify({
                            __type: 'InvalidParameterException',
                            message: 'Invalid instance ID'
                        }));
                        return;
                    }
                    
                    const queueIds = filters?.Queues || [];
                    const metrics = generateMetrics(queueIds);
                    
                    const response = {
                        MetricResults: metrics
                    };
                    
                    res.writeHead(200, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify(response));
                    
                } else if (action === 'Connect_20170801.ListAgentStatuses') {
                    console.log('Handling ListAgentStatuses request');
                    
                    const instanceId = requestData.InstanceId;
                    if (!instanceId || !instanceId.includes(INSTANCE_ID)) {
                        res.writeHead(400, { 'Content-Type': 'application/json' });
                        res.end(JSON.stringify({
                            __type: 'InvalidParameterException',
                            message: 'Invalid instance ID'
                        }));
                        return;
                    }
                    
                    const response = {
                        AgentStatusSummaryList: MOCK_AGENT_STATUSES
                    };
                    
                    res.writeHead(200, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify(response));
                    
                } else {
                    console.log('Unknown action:', action);
                    res.writeHead(400, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify({
                        __type: 'InvalidActionException',
                        message: 'Unknown action: ' + action
                    }));
                }
            } catch (e) {
                console.error('Error processing request:', e);
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({
                    __type: 'InternalServiceException',
                    message: 'Internal server error'
                }));
            }
        });
        return;
    }
    
    // Default 404
    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Not found' }));
});

server.listen(PORT, () => {
    console.log(`AWS Connect Mock API Server running on http://localhost:${PORT}`);
    console.log(`Instance ID: ${INSTANCE_ID}`);
    console.log(`Error rate: ${errorRate * 100}%`);
    console.log('');
    console.log('Available endpoints:');
    console.log('  GET  /health              - Health check');
    console.log('  POST /                    - AWS Connect API (with x-amz-target header)');
    console.log('  POST /admin/errorRate     - Set error simulation rate');
    console.log('');
    console.log('Example usage:');
    console.log('  curl http://localhost:3000/health');
    console.log('  curl -X POST http://localhost:3000/admin/errorRate -d \'{"rate": 0.1}\'');
});