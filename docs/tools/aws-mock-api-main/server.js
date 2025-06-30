/**
 * @file This file implements a mock AWS Connect API server for testing and
 * development purposes. It simulates a subset of the AWS Connect API, allowing
 * the exporter to be tested without connecting to a real AWS instance.
 * @author Sabio
 * @license ISC
 */

const express = require('express');
const bodyParser = require('body-parser');

const app = express();

// Parse JSON bodies for POST requests
app.use(bodyParser.json());

// Add CORS headers for local testing
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Headers', '*');
    next();
});

/****************************************************
 * CONSTANTS AND CONFIGURATION
 ****************************************************/

// Mock AWS Connect instance and account details - overridable via environment variables
const INSTANCE_ID = process.env.MOCK_INSTANCE_ID || 'f604177c-b2ab-4c76-9033-635d195b2772';
const ACCOUNT_ID  = process.env.MOCK_ACCOUNT_ID  || '123456789012';
const REGION      = process.env.MOCK_REGION      || 'eu-west-2';

// Error simulation controls (can be overridden via MOCK_ERROR_RATE)
let errorRate = 0.0; // default to no errors for stable tests
if (process.env.MOCK_ERROR_RATE) {
    const envErrorRate = parseFloat(process.env.MOCK_ERROR_RATE);
    if (!isNaN(envErrorRate) && envErrorRate >= 0 && envErrorRate <= 1) {
        errorRate = envErrorRate;
    } else {
        console.warn(`Invalid MOCK_ERROR_RATE: ${process.env.MOCK_ERROR_RATE}. Using default: ${errorRate}`);
    }
}
let currentErrors = 0;          // Counter for consecutive errors
let maxConsecutiveErrors = 2;   // Max consecutive errors to allow before resetting

// Pagination configuration (can be overridden via ITEMS_PER_PAGE)
let itemsPerPage = 1000;  // Default to high value to avoid pagination issues
if (process.env.ITEMS_PER_PAGE) {
    const envItemsPerPage = parseInt(process.env.ITEMS_PER_PAGE, 10);
    if (!isNaN(envItemsPerPage) && envItemsPerPage > 0) {
        itemsPerPage = envItemsPerPage;
    } else {
        console.warn(`Invalid ITEMS_PER_PAGE: ${process.env.ITEMS_PER_PAGE}. Using default: ${itemsPerPage}`);
    }
}
const ITEMS_PER_PAGE = itemsPerPage;

// ----------------------------------------------
// Metric configuration
// ----------------------------------------------
const fs = require('fs');
const path = require('path');

const CONFIG_PATH = path.join(__dirname, 'metrics-config.json');
let metricsConfig;
try {
  metricsConfig = JSON.parse(fs.readFileSync(CONFIG_PATH, 'utf8'));
  console.log(`Loaded metric config from ${CONFIG_PATH}`);
} catch (err) {
  console.warn(`Could not load ${CONFIG_PATH}: ${err}. Using built-in defaults`);
  metricsConfig = {
    metrics: [
      { name: 'CONTACTS_IN_QUEUE', unit: 'COUNT', type: 'random', min: 1, max: 8 },
      { name: 'AGENTS_STAFFED', unit: 'COUNT', type: 'random', min: 10, max: 25 },
      { name: 'AGENTS_ON_CONTACT', unit: 'COUNT', type: 'ratio', base: 'AGENTS_STAFFED', ratio: 0.6 },
      { name: 'AGENTS_AVAILABLE', unit: 'COUNT', type: 'difference', base: 'AGENTS_STAFFED', subtract: 'AGENTS_ON_CONTACT' },
      { name: 'OLDEST_CONTACT_AGE', unit: 'SECONDS', type: 'ratio', base: 'CONTACTS_IN_QUEUE', ratio: 15 },
      { name: 'SLOTS_ACTIVE', unit: 'COUNT', type: 'ratio', base: 'AGENTS_STAFFED', ratio: 1.0 },
      { name: 'SLOTS_AVAILABLE', unit: 'COUNT', type: 'difference', base: 'SLOTS_ACTIVE', subtract: 'CONTACTS_IN_QUEUE' },
      { name: 'CONTACTS_QUEUED', unit: 'COUNT', type: 'cumulative', stepMin: 0, stepMax: 5 },
      { name: 'CONTACTS_ABANDONED', unit: 'COUNT', type: 'cumulative', stepMin: 0, stepMax: 3 },
      { name: 'CONTACTS_HANDLED', unit: 'COUNT', type: 'cumulative', stepMin: 0, stepMax: 5 }
    ]
  };
}

const VALID_METRICS = {};
const MetricGenerators = { baseMetrics: {}, derivedMetrics: {} };

function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

metricsConfig.metrics.forEach(def => {
  VALID_METRICS[def.name] = def.unit || 'COUNT';
  if (def.type === 'random') {
    MetricGenerators.baseMetrics[def.name] = () => ({
      value: randomInt(def.min, def.max),
      unit: def.unit
    });
  } else if (def.type === 'cumulative') {
    MetricGenerators.baseMetrics[def.name] = (queue, baseValues, d) => {
      if (!queueTotals[queue.Id]) queueTotals[queue.Id] = {};
      if (!queueTotals[queue.Id][def.name]) queueTotals[queue.Id][def.name] = 0;
      queueTotals[queue.Id][def.name] += randomInt(def.stepMin, def.stepMax);
      return { value: queueTotals[queue.Id][def.name], unit: def.unit };
    };
  } else if (def.type === 'ratio') {
    MetricGenerators.derivedMetrics[def.name] = (queue, baseValues, d) => {
      const base = baseValues[def.base] ? baseValues[def.base].value : 0;
      return { value: Math.floor(base * def.ratio), unit: def.unit };
    };
  } else if (def.type === 'difference') {
    MetricGenerators.derivedMetrics[def.name] = (queue, baseValues, d) => {
      const base = baseValues[def.base] ? baseValues[def.base].value : 0;
      const sub = baseValues[def.subtract] ? baseValues[def.subtract].value : 0;
      return { value: Math.max(0, base - sub), unit: def.unit };
    };
  }
});

// Valid channels that this mock service recognizes
const VALID_CHANNELS = ['VOICE', 'CHAT', 'TASK'];

/****************************************************
 * AWS-LIKE ERRORS
 ****************************************************/
// Mock error objects that simulate the structure of
// various AWS service exceptions.
const AWS_ERRORS = {
  THROTTLING: {
    __type: 'ThrottlingException',
    message: 'Rate exceeded'
  },
  SERVICE_UNAVAILABLE: {
    __type: 'ServiceUnavailableException',
    message: 'Service is unavailable'
  },
  INTERNAL_SERVICE_ERROR: {
    __type: 'InternalServiceErrorException',
    message: 'Internal server error occurred'
  },
  RESOURCE_NOT_FOUND: {
    __type: 'ResourceNotFoundException',
    message: 'The specified resource was not found'
  },
  INVALID_PARAMETER: {
    __type: 'InvalidParameterException',
    message: 'One or more parameters are not valid'
  },
  VALIDATION_EXCEPTION: {
    __type: 'ValidationException',
    message: 'Validation error occurred'
  }
};

/****************************************************
 * MOCK QUEUES
 * 
 * This object simulates various queue configurations
 * as if they were in an AWS Connect instance.
 ****************************************************/
// Minimal queue summary list returned by the ListQueues endpoint
// to facilitate queue discovery in tests.
const MOCK_QUEUE_SUMMARIES = [
  { Id: 'q-1', Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/queue/q-1`, Name: 'Sales_Queue_VOICE' },
  { Id: 'q-2', Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/queue/q-2`, Name: 'Support_Queue_CHAT' },
  { Id: 'q-3', Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/queue/q-3`, Name: 'Test_Queue_TASK' }
];

// Simplified routing profile definitions for queue lookups
const ROUTING_PROFILES = {
  'rp-sales': {
    Id: 'rp-sales',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/routing-profile/rp-sales`,
    Queues: ['queue-sales-01']
  },
  'rp-support': {
    Id: 'rp-support',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/routing-profile/rp-support`,
    Queues: ['queue-support-01']
  },
  'tf-support': {
    Id: 'tf-support',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/routing-profile/tf-support`,
    Queues: ['queue-support-999', 'queue-chat-999', 'queue-task-999']
  }
};

const MOCK_QUEUES = {
  'sales': {
    Id: 'queue-sales-01',
    Name: 'Sales Queue',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/queue/queue-sales-01`,
    QueueType: 'STANDARD',
    RoutingProfile: 'rp-sales',
    StaffingTarget: { min: 10, max: 25 },
    Channel: 'VOICE',
    Priority: 1,
    HoursOfOperation: 'BasicHours',
    OutboundCallerId: '+1234567890'
  },
  'support': {
    Id: 'queue-support-01',
    Name: 'Support Queue',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/queue/queue-support-01`,
    QueueType: 'STANDARD',
    RoutingProfile: 'rp-support',
    StaffingTarget: { min: 15, max: 30 },
    Channel: 'VOICE',
    Priority: 2,
    HoursOfOperation: 'ExtendedHours',
    OutboundCallerId: '+678912345'
  },
  'monitoring': {
    Id: 'queue-support-999',
    Name: 'Monitoring Queue',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/queue/queue-monitoring-01`,
    QueueType: 'AGENT',
    RoutingProfile: 'tf-support',
    StaffingTarget: { min: 5, max: 15 },
    Channel: 'VOICE',
    Priority: 2,
    HoursOfOperation: 'ExtendedHours',
    OutboundCallerId: '+987654321'
  },
  'monitoring_chat': {
    Id: 'queue-chat-999',
    Name: 'Monitoring Chat Queue',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/queue/queue-monitoring-chat-01`,
    QueueType: 'AGENT',
    RoutingProfile: 'tf-support',
    StaffingTarget: { min: 20, max: 70 },
    Channel: 'CHAT',
    Priority: 2,
    HoursOfOperation: 'ExtendedHours',
    OutboundCallerId: '+987654321'
  },
  'monitoring_task': {
    Id: 'queue-task-999',
    Name: 'Monitoring Task Queue',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/queue/queue-monitoring-task-01`,
    QueueType: 'AGENT',
    RoutingProfile: 'tf-support',
    StaffingTarget: { min: 10, max: 25 },
    Channel: 'TASK',
    Priority: 2,
    HoursOfOperation: 'ExtendedHours',
    OutboundCallerId: '+987654321'
  }
};

// ----------------------------------------------------------
// Queue metric totals
// ----------------------------------------------------------
// Track cumulative historical metrics per queue so that successive
// requests return increasing values similar to the real API.
const queueTotals = {};
Object.values(MOCK_QUEUES).forEach(q => {
  queueTotals[q.Id] = {
    CONTACTS_QUEUED: 0,
    CONTACTS_ABANDONED: 0,
    CONTACTS_HANDLED: 0,
    CONTACTS_HANDLED_OUTBOUND: 0
  };
});

/****************************************************
 * MOCK AGENT STATUSES
 *
 * These entries simulate agent status objects returned
 * by the AWS Connect ListAgentStatuses API.
 ****************************************************/
// List of all possible agent statuses. This mimics the static configuration in
// a real Connect instance where admins define available states.
const agentStatuses = [
  { Id: 'available-status-id', Name: 'Available', Type: 'ROUTABLE' },
  { Id: 'acw-status-id', Name: 'After Contact Work', Type: 'NOT_ROUTABLE' },
  { Id: 'oncall-status-id', Name: 'On Call', Type: 'NOT_ROUTABLE' },
  { Id: 'lunch-status-id', Name: 'Lunch Break', Type: 'CUSTOM' },
  { Id: 'error-status-id', Name: 'Error', Type: 'NOT_ROUTABLE' }
];

// Dynamic roster of agents. Each agent references one of the statuses above via
// `currentStatusId`. This array will be mutated by the simulation logic below.
let agentRoster = [
  { id: 'agent-1', username: 'jdoe', currentStatusId: 'available-status-id' },
  { id: 'agent-2', username: 'psmith', currentStatusId: 'available-status-id' },
  { id: 'agent-3', username: 'rroe', currentStatusId: 'lunch-status-id' },
  { id: 'agent-4', username: 'mday', currentStatusId: 'available-status-id' }
];

// Internal counter for generating unique agent IDs when new agents log in.
let nextAgentId = 5;

// ----------------------------------------------------------
// Simulation Helpers
// ----------------------------------------------------------

// Randomly update agent statuses to simulate activity.
function simulateAgentStateChanges() {
  agentRoster.forEach(agent => {
    // 30% chance an agent changes state each interval
    if (Math.random() < 0.3) {
      const currentIndex = agentStatuses.findIndex(s => s.Id === agent.currentStatusId);
      let newIndex;
      do {
        newIndex = Math.floor(Math.random() * agentStatuses.length);
      } while (newIndex === currentIndex);

      const newStatus = agentStatuses[newIndex];
      console.log(`SIMULATOR: Agent ${agent.username} changed state to ${newStatus.Name}`);
      agent.currentStatusId = newStatus.Id;
    }
  });
}

// Periodically adjust the agent roster: agents logging in and out.
function simulateRosterChanges() {
  // 10% chance an agent logs out (keep at least two)
  if (Math.random() < 0.1 && agentRoster.length > 2) {
    const index = Math.floor(Math.random() * agentRoster.length);
    const leaving = agentRoster[index];
    console.log(`SIMULATOR: Agent ${leaving.username} has logged out.`);
    agentRoster.splice(index, 1);
  }

  // 15% chance a new agent logs in (cap at 10 agents)
  if (Math.random() < 0.15 && agentRoster.length < 10) {
    const newAgent = {
      id: `agent-${nextAgentId}`,
      username: `newuser${nextAgentId}`,
      currentStatusId: 'available-status-id'
    };
    console.log(`SIMULATOR: Agent ${newAgent.username} has logged in.`);
    agentRoster.push(newAgent);
    nextAgentId++;
  }
}

// Kick off the simulations
setInterval(simulateAgentStateChanges, 5000);   // status changes every 5s
setInterval(simulateRosterChanges, 15000);      // roster changes every 15s

/****************************************************
 * MOCK USERS
 *
 * Minimal set of users returned by the GetCurrentUserData API
 * so that realtime agent status metrics can be collected.
 ****************************************************/
// Enhanced mock users for realistic dashboard testing
const MOCK_USERS = [
  // Available agents (5 users)
  {
    Id: '6aed748d-3a99-410c-9e3a-afe3fda0af82',
    Username: 'alice.smith',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/6aed748d-3a99-410c-9e3a-afe3fda0af82`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['rp-sales'].Arn,
      Id: 'rp-sales'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-001`,
      StatusName: 'Available'
    }
  },
  {
    Id: '762903b8-0d23-4076-8b5f-8d8f633161fe',
    Username: 'bob.jones',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/762903b8-0d23-4076-8b5f-8d8f633161fe`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['rp-sales'].Arn,
      Id: 'rp-sales'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-001`,
      StatusName: 'Available'
    }
  },
  {
    Id: 'a3c6108c-07de-4c1b-86c4-8f54970e3ae8',
    Username: 'carol.davis',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/a3c6108c-07de-4c1b-86c4-8f54970e3ae8`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['rp-sales'].Arn,
      Id: 'rp-sales'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-001`,
      StatusName: 'Available'
    }
  },
  {
    Id: 'beb7cc01-cb59-44ec-abf5-4092c7eb6375',
    Username: 'david.wilson',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/beb7cc01-cb59-44ec-abf5-4092c7eb6375`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['rp-sales'].Arn,
      Id: 'rp-sales'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-001`,
      StatusName: 'Available'
    }
  },
  {
    Id: 'ad6c7251-66ef-4ab0-8b3b-23383cf6af3d',
    Username: 'emma.brown',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/ad6c7251-66ef-4ab0-8b3b-23383cf6af3d`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['rp-sales'].Arn,
      Id: 'rp-sales'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-001`,
      StatusName: 'Available'
    }
  },
  // Agents on contact (3 users)
  {
    Id: '15d897b4-ce68-4cb8-aa5b-4e34baeca3dc',
    Username: 'frank.taylor',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/15d897b4-ce68-4cb8-aa5b-4e34baeca3dc`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['rp-support'].Arn,
      Id: 'rp-support'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-002`,
      StatusName: 'On Call'
    }
  },
  {
    Id: '55a65983-30c8-4492-bb07-708a9c30d729',
    Username: 'grace.anderson',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/55a65983-30c8-4492-bb07-708a9c30d729`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['rp-support'].Arn,
      Id: 'rp-support'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-002`,
      StatusName: 'On Call'
    }
  },
  {
    Id: '47a61b5c-093d-4e16-ad5a-29a9fd686578',
    Username: 'henry.martinez',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/47a61b5c-093d-4e16-ad5a-29a9fd686578`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['rp-support'].Arn,
      Id: 'rp-support'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-002`,
      StatusName: 'On Call'
    }
  },
  // Agents in after contact work (2 users)
  {
    Id: '9e0a4a18-5a7c-46e8-a724-40647f359f67',
    Username: 'iris.garcia',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/9e0a4a18-5a7c-46e8-a724-40647f359f67`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['tf-support'].Arn,
      Id: 'tf-support'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-003`,
      StatusName: 'After Contact Work'
    }
  },
  {
    Id: 'e4a4da2c-743d-429d-86de-22a7ff19b0c7',
    Username: 'jack.rodriguez',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/e4a4da2c-743d-429d-86de-22a7ff19b0c7`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['tf-support'].Arn,
      Id: 'tf-support'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-003`,
      StatusName: 'After Contact Work'
    }
  },
  // Agents on break (1 user)
  {
    Id: '4ebb98ea-bb7a-4a80-aaaf-e99f176ab802',
    Username: 'kelly.lopez',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/4ebb98ea-bb7a-4a80-aaaf-e99f176ab802`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['tf-support'].Arn,
      Id: 'tf-support'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-004`,
      StatusName: 'Lunch Break'
    }
  },
  // Agent with error status
  {
    Id: 'b8f7c12e-4d3a-4f5c-bc23-1a8e7f9d2c4e',
    Username: 'mike.wilson',
    Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent/b8f7c12e-4d3a-4f5c-bc23-1a8e7f9d2c4e`,
    RoutingProfile: {
      Arn: ROUTING_PROFILES['tf-support'].Arn,
      Id: 'tf-support'
    },
    Status: {
      StatusArn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/status-005`,
      StatusName: 'Error'
    }
  }
];

/****************************************************
 * SOPHISTICATED METRIC GENERATION
 * 
 * We separate metrics into baseMetrics and 
 * derivedMetrics for clarity and reusability.
 ****************************************************/
class MetricGenerator {
  constructor(queue) {
    this.queue = queue;
    this.baseValues = {};
  }

  generateMetrics(metricNames) {
    // First generate base metrics
    for (const name of metricNames) {
      if (MetricGenerators.baseMetrics[name]) {
        this.baseValues[name] = MetricGenerators.baseMetrics[name](this.queue, this.baseValues, {});
      }
    }

    // Then generate derived metrics
    const results = [];
    for (const name of metricNames) {
      let metricValue;
      if (MetricGenerators.baseMetrics[name]) {
        metricValue = this.baseValues[name];
      } else if (MetricGenerators.derivedMetrics[name]) {
        metricValue = MetricGenerators.derivedMetrics[name](this.queue, this.baseValues, {});
      }

      if (metricValue) {
        results.push({
          Dimensions: {
            Queue: {
              Arn: this.queue.Arn,
              Id: this.queue.Id,
              Name: this.queue.Name
            },
            Channel: this.queue.Channel
          },
          Collections: [{
            Metric: {
              Name: name,
              Unit: metricValue.unit
            },
            Value: metricValue.value
          }]
        });
      }
    }

    return results;
  }
}

// Helper for historical metrics (GetMetricDataV2)
function generateMetricsV2(queue, metricDefs) {
  const baseValues = {};
  for (const def of metricDefs) {
    const name = def.Name;
    if (MetricGenerators.baseMetrics[name]) {
      baseValues[name] = MetricGenerators.baseMetrics[name](queue, baseValues, def);
    }
  }

  const results = [];
  for (const def of metricDefs) {
    const name = def.Name;
    let metricValue;
    if (MetricGenerators.baseMetrics[name]) {
      metricValue = baseValues[name];
    } else if (MetricGenerators.derivedMetrics[name]) {
      metricValue = MetricGenerators.derivedMetrics[name](queue, baseValues, def);
    }

    if (metricValue) {
      const metric = { Name: name, Unit: metricValue.unit };
      if (def.MetricFilters) {
        metric.MetricFilters = def.MetricFilters;
      }
      results.push({
        Dimensions: { QUEUE: queue.Id, CHANNEL: queue.Channel },
        Collections: [{ Metric: metric, Value: metricValue.value }]
      });
    }
  }

  return results;
}

// Metric generators for different metric types are now
// built dynamically from the configuration above.

/****************************************************
 * VALIDATION FUNCTIONS
 ****************************************************/

/**
 * validateMetrics(metrics)
 * Checks if metrics is an array and that each metric 
 * matches one of the valid metric names in VALID_METRICS.
 * 
 * @param {Array} metrics - Array of metric definitions
 * @return {boolean} true if valid, false otherwise
 */
function validateMetrics(metrics) {
    if (!Array.isArray(metrics)) {
        console.log('Metrics is not an array');
        return false;
    }
    
    const validationResults = metrics.every(metric => {
        console.log('Checking metric:', JSON.stringify(metric));
        const isValid = metric && 
            typeof metric.Name === 'string' && 
            VALID_METRICS[metric.Name] !== undefined;
        console.log('Metric validation result:', isValid);
        return isValid;
    });
  
    console.log('Valid metrics:', validationResults);
    return validationResults;
}

/**
 * validateFilters(filters)
 * Ensures that each filter has a valid array of queue IDs
 * and optionally valid channel information (single Channel or
 * Channels array).
 * 
 * @param {Array} filters - Array of filter objects
 * @return {boolean} true if valid, false otherwise
 */
// Validate filters for GetMetricDataV2 (FilterV2 structure)
function validateFiltersV2(filters) {
    console.log('\n=== Starting FilterV2 Validation ===');
    console.log('Validating filters:', JSON.stringify(filters, null, 2));

    // 1. Basic structure validation
    if (!Array.isArray(filters)) {
        console.log('❌ Invalid filters - should be an array:', typeof filters);
        return false;
    }

    // 2. Find QUEUE filter and validate
    const queueFilter = filters.find(f => f.FilterKey === "QUEUE");
    if (!queueFilter || !Array.isArray(queueFilter.FilterValues) || queueFilter.FilterValues.length === 0) {
        console.log('❌ Invalid or missing QUEUE filter:', queueFilter);
        return false;
    }

    console.log('\n📋 Available Queues:');
    Object.values(MOCK_QUEUES).forEach(q => {
        console.log(`  - ID: ${q.Id}, Name: ${q.Name}`);
    });

    // 3. Validate each Queue ID
    const validQueues = queueFilter.FilterValues.every(queueId => {
        console.log(`\n🔍 Validating Queue ID: ${queueId}`);
        
        const queue = Object.values(MOCK_QUEUES).find(q => q.Id === queueId);
        const found = !!queue;

        if (found) {
            console.log('✅ Found matching queue:', {
                Id: queue.Id,
                Name: queue.Name,
                Type: queue.QueueType,
                Channel: queue.Channel
            });
        } else {
            console.log('❌ No matching queue found for ID:', queueId);
            console.log('Available Queue IDs:', Object.values(MOCK_QUEUES).map(q => q.Id));
        }

        return found;
    });

    let channelsValid = true;

    // 4. Validate CHANNEL filter if present
    const channelFilter = filters.find(f => f.FilterKey === "CHANNEL");
    if (channelFilter) {
        console.log('\n📡 Validating Channel filter:', channelFilter.FilterValues);
        channelsValid = channelFilter.FilterValues.every(ch => VALID_CHANNELS.includes(ch));
        if (!channelsValid) {
            console.log('❌ Invalid channels. Valid channels are:', VALID_CHANNELS);
            return false;
        }
        console.log('✅ Channel validation passed');
    }

    // 5. Final result
    console.log('\n=== FilterV2 Validation Result ===');
    const result = validQueues && channelsValid;
    console.log(result ? '✅ All validations passed' : '❌ Validation failed');

    return result;
}

// Validate filters for GetCurrentMetricData (legacy structure)
function validateFilters(filters) {
    console.log('\n=== Starting Legacy Filter Validation ===');
    console.log('Validating filters:', JSON.stringify(filters, null, 2));

    // 1. Basic structure validation
    if (!filters || typeof filters !== 'object') {
        console.log('❌ Invalid filters object - not an object:', typeof filters);
        return false;
    }

    // 2. Validate Queues array
    if (!Array.isArray(filters.Queues) || filters.Queues.length === 0) {
        console.log('❌ Invalid or empty Queues array:', filters.Queues);
        return false;
    }

    console.log('\n📋 Available Queues:');
    Object.values(MOCK_QUEUES).forEach(q => {
        console.log(`  - ID: ${q.Id}, Name: ${q.Name}`);
    });

    // 3. Validate each Queue ID
    const validQueues = filters.Queues.every(queueId => {
        console.log(`\n🔍 Validating Queue ID: ${queueId}`);
        
        const queue = Object.values(MOCK_QUEUES).find(q => q.Id === queueId);
        const found = !!queue;

        if (found) {
            console.log('✅ Found matching queue:', {
                Id: queue.Id,
                Name: queue.Name,
                Type: queue.QueueType,
                Channel: queue.Channel
            });
        } else {
            console.log('❌ No matching queue found for ID:', queueId);
            console.log('Available Queue IDs:', Object.values(MOCK_QUEUES).map(q => q.Id));
        }

        return found;
    });

    let channelsValid = true;

    // 4a. Single channel validation (legacy)
    if (filters.Channel) {
        console.log('\n📡 Validating Channel:', filters.Channel);
        channelsValid = VALID_CHANNELS.includes(filters.Channel);
        if (!channelsValid) {
            console.log('❌ Invalid channel. Valid channels are:', VALID_CHANNELS);
            return false;
        }
        console.log('✅ Channel validation passed');
    }

    // 4b. Channels array validation (new style)
    if (Array.isArray(filters.Channels)) {
        console.log('\n📡 Validating Channels array:', filters.Channels);
        channelsValid = filters.Channels.every(ch => VALID_CHANNELS.includes(ch));
        if (!channelsValid) {
            console.log('❌ Invalid channels. Valid channels are:', VALID_CHANNELS);
            return false;
        }
        console.log('✅ Channels validation passed');
    }

    // 5. Final result
    console.log('\n=== Legacy Filter Validation Result ===');
    const result = validQueues && channelsValid;
    console.log(result ? '✅ All validations passed' : '❌ Validation failed');

    return result;
}

/****************************************************
 * ERROR HANDLING/ SIMULATION FUNCTIONS
 ****************************************************/

/**
 * shouldError()
 * Randomly decides if an error should be triggered 
 * based on errorRate and maxConsecutiveErrors.
 * 
 * @return {Object|null} - AWS-like error object or null
 */
function shouldError() {
  // Use the errorRate to decide if we should trigger an error
  if (Math.random() < errorRate) {
    currentErrors++;
    if (currentErrors <= maxConsecutiveErrors) {
      // Randomly select an error from AWS_ERRORS
      const errors = Object.values(AWS_ERRORS);
      return errors[Math.floor(Math.random() * errors.length)];
    }
  }
  // Reset consecutive error counter if we didn't error this time
  currentErrors = 0;
  return null;
}

/****************************************************
 * PAGINATION HELPER
 ****************************************************/

/**
 * paginateResults(items, nextToken)
 * Splits the items into pages of ITEMS_PER_PAGE each.
 * If nextToken is provided, pagination begins from 
 * that index.
 * 
 * @param {Array} items - The full array of items
 * @param {string} nextToken - The current pagination token
 * @return {Object} { items, nextToken }
 */
// Update the paginateResults function
function paginateResults(items, nextToken) {
    // Parse nextToken as number, default to 0
    const startIndex = nextToken ? parseInt(nextToken, 10) : 0;
    
    // Debug pagination
    console.log(`Pagination request: items.length=${items.length}, nextToken=${nextToken}, startIndex=${startIndex}`);
    
    // If startIndex is beyond array length, return empty result
    if (startIndex >= items.length) {
        console.log('Pagination: startIndex beyond array length, returning empty result');
        return {
            items: [],
            nextToken: null
        };
    }
    
    const endIndex = Math.min(startIndex + ITEMS_PER_PAGE, items.length);
    const hasMore = endIndex < items.length;

    console.log(`Pagination: startIndex=${startIndex}, endIndex=${endIndex}, hasMore=${hasMore}`);
    
    return {
        items: items.slice(startIndex, endIndex),
        nextToken: hasMore ? endIndex.toString() : null
    };
}

/****************************************************
 * MAIN ROUTE HANDLER
 ****************************************************/
// Express 5.x no longer supports using `*` as a catch-all route
// and will throw a `Missing parameter name` error via path-to-regexp.
// Use a named wildcard parameter instead which also matches the root path.
app.all('/{*path}', (req, res, next) => {
    console.log('\n=== Incoming Request ===');
    console.log('Method:', req.method);
    console.log('Path:', req.path);
    console.log('Headers:', JSON.stringify(req.headers, null, 2));
    console.log('Query:', JSON.stringify(req.query, null, 2));
    console.log('Body:', JSON.stringify(req.body, null, 2));

    const action = req.headers['x-amz-target'];
    console.log('Action:', action);

    // Check if this is an AWS SDK style request
    if (action === 'ListQueues' || req.path.startsWith('/queues-summary/')) {
        // Extract instanceId from either query or path
        let instanceId;
        if (req.path.startsWith('/queues-summary/')) {
            instanceId = req.path.split('/queues-summary/')[1]?.split('?')[0];
            console.log('Extracted instanceId from path:', instanceId);
        } else {
            instanceId = req.body.InstanceId || req.query.InstanceId;
            console.log('Using instanceId from request:', instanceId);
        }
        
        // Validate instanceId
        if (!instanceId) {
            console.log('Missing instanceId');
            return res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
        }

        if (instanceId !== INSTANCE_ID) {
            console.log('Instance ID mismatch:', instanceId, '!==', INSTANCE_ID);
            return res.status(404).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
        }

        // Validate nextToken if provided
        const nextToken = req.query.nextToken;
        if (nextToken && isNaN(parseInt(nextToken, 10))) {
            console.log('Invalid nextToken:', nextToken);
            return res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
        }

        // Simulate random errors
        const error = shouldError();
        if (error) {
            console.log('Random error triggered:', error);
            return res.status(500).json(error);
        }

        // Get queue list and paginate
        const queueList = Object.values(MOCK_QUEUES);
        const { items, nextToken: newNextToken } = paginateResults(queueList, nextToken);

        console.log('Returning queues:', items);
        console.log('Next token:', newNextToken);
        
        return res.json({
            QueueSummaryList: items,
            NextToken: newNextToken
        });
    }
    else if (action === 'ListUsers' || req.path.startsWith('/users-summary/')) {
        console.log('\n=== Processing List Users Request ===');
        
        // Extract instanceId from either query or path
        let instanceId;
        if (req.path.startsWith('/users-summary/')) {
            instanceId = req.path.split('/users-summary/')[1]?.split('?')[0];
            console.log('📍 Extracted instanceId from path:', instanceId);
        } else {
            instanceId = req.body.InstanceId || req.query.InstanceId;
            console.log('📍 Using instanceId from request:', instanceId);
        }

        // Validate instanceId
        if (!instanceId) {
            console.log('❌ Missing instanceId');
            return res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
        }

        if (instanceId !== INSTANCE_ID) {
            console.log('❌ Instance ID mismatch:', instanceId, '!==', INSTANCE_ID);
            return res.status(404).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
        }

        // Validate nextToken if provided
        const nextToken = req.query.nextToken || (req.body && req.body.NextToken);
        if (nextToken && isNaN(parseInt(nextToken, 10))) {
            console.log('❌ Invalid nextToken:', nextToken);
            return res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
        }

        // Simulate random errors
        const error = shouldError();
        if (error) {
            console.log('⚠️ Random error triggered:', error);
            return res.status(500).json(error);
        }

        // Convert MOCK_USERS to AWS Connect UserSummary format
        // Using pagination with smaller page size (4 users) to exercise pagination
        const userSummaries = MOCK_USERS.map(user => ({
            Id: user.Id,
            Username: user.Username,
            Arn: user.Arn,
            RoutingProfileId: user.RoutingProfile.Id,
            RoutingProfileArn: user.RoutingProfile.Arn,
            HierarchyGroupId: null, // Not implemented in mock
            Tags: {} // Not implemented in mock
        }));

        // Use custom pagination with page size of 4 to test pagination with 11 users
        const pageSize = 4;
        const startIndex = nextToken ? parseInt(nextToken, 10) : 0;
        const endIndex = Math.min(startIndex + pageSize, userSummaries.length);
        const items = userSummaries.slice(startIndex, endIndex);
        const newNextToken = endIndex < userSummaries.length ? endIndex.toString() : null;

        console.log(`📑 Paginating users: ${startIndex}-${endIndex-1} of ${userSummaries.length} total`);
        console.log('📤 Returning users:', items.map(u => u.Username));
        console.log('🔗 Next token:', newNextToken);

        return res.json({
            UserSummaryList: items,
            NextToken: newNextToken,
            ApproximateTotalCount: userSummaries.length
        });
    } 
    else if (action === 'GetCurrentMetricData' || req.path.startsWith('/metrics/current/')) {
        console.log('\n=== Processing Metrics Request ===');
        
        // 1. Extract and validate instanceId
        let instanceId;
        if (req.path.startsWith('/metrics/current/')) {
            instanceId = req.path.split('/metrics/current/')[1]?.split('?')[0];
            console.log('📍 Extracted instanceId from path:', instanceId);
        } else {
            instanceId = req.body.InstanceId;
            console.log('📍 Using instanceId from body:', instanceId);
        }

        if (!instanceId || instanceId !== INSTANCE_ID) {
            console.log('❌ Invalid instanceId:', instanceId);
            console.log('Expected:', INSTANCE_ID);
            return res.status(404).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
        }

        try {
            const body = req.body;
            console.log('\n📦 Request body:', JSON.stringify(body, null, 2));
            
            // 2. Validate request structure
            if (!body.Filters || !body.CurrentMetrics) {
                console.log('❌ Missing required fields. Filters or CurrentMetrics not found');
                return res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
            }

            // 3. Validate metrics
            console.log('\n🔍 Validating metrics...');
            const metricsValid = validateMetrics(body.CurrentMetrics);
            if (!metricsValid) {
                console.log('❌ Metric validation failed');
                return res.status(400).json(AWS_ERRORS.VALIDATION_EXCEPTION);
            }
            console.log('✅ Metrics validation passed');

            // 4. Validate filters
            console.log('\n🔍 Validating filters...');
            const filtersValid = validateFilters(body.Filters);
            if (!filtersValid) {
                console.log('❌ Filter validation failed');
                return res.status(400).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
            }
            console.log('✅ Filters validation passed');

            // 5. Check for random errors
            const error = shouldError();
            if (error) {
                console.log('⚠️ Random error triggered:', error);
                return res.status(500).json(error);
            }

            // 6. Generate metrics
            console.log('\n📊 Generating metrics...');
            const allMetricResults = [];
            const queueIds = body.Filters.Queues;

            const channelFilters = [];
            if (body.Filters.Channel) {
                channelFilters.push(body.Filters.Channel);
            }
            if (Array.isArray(body.Filters.Channels)) {
                channelFilters.push(...body.Filters.Channels);
            }

            queueIds.forEach(queueId => {
                const queue = Object.values(MOCK_QUEUES).find(q => q.Id === queueId);
                if (queue) {
                    if (channelFilters.length > 0 && !channelFilters.includes(queue.Channel)) {
                        console.log(`Skipping queue ${queue.Id} due to channel filter`);
                        return;
                    }
                    console.log(`Generating metrics for queue: ${queue.Name} (${queue.Id})`);
                    const generator = new MetricGenerator(queue);
                    const metrics = generator.generateMetrics(
                        body.CurrentMetrics.map(m => m.Name)
                    );
                    console.log('Generated metrics:', JSON.stringify(metrics, null, 2));
                    allMetricResults.push(...metrics);
                }
            });

            // 7. Paginate results
            console.log('\n📑 Paginating results...');
            const { items, nextToken: newNextToken } = paginateResults(
                allMetricResults, 
                body.NextToken
            );

            // 8. Send response
            const response = {
                MetricResults: items,
                NextToken: newNextToken,
                DataSnapshotTime: Date.now()
            };

            console.log('\n✅ Sending response:', JSON.stringify(response, null, 2));
            return res.json(response);

        } catch (error) {
            console.error('\n❌ Internal error:', error);
            return res.status(500).json(AWS_ERRORS.INTERNAL_SERVICE_ERROR);
        }
    }
    else if (action === 'GetMetricData') {
        // Simple historical metrics endpoint used by unit tests
        const groupings = req.body.Groupings || [];
        let response;
        if (groupings.includes('QUEUE') && groupings.includes('CHANNEL')) {
            response = {
                MetricResults: [{
                    Dimensions: { QUEUE: 'Sales_Queue_VOICE', CHANNEL: 'VOICE' },
                    Collections: [{ Metric: { Name: 'CONTACTS_HANDLED' }, Value: 150.0 }]
                }]
            };
        } else if (groupings.includes('QUEUE')) {
            response = {
                MetricResults: [{
                    Dimensions: { QUEUE: 'Sales_Queue_VOICE' },
                    Collections: [{ Metric: { Name: 'CONTACTS_HANDLED' }, Value: 250.0 }]
                }]
            };
        } else {
            response = { MetricResults: [] };
        }
        return res.json(response);
    }
    else if (action === 'GetMetricDataV2' || req.path.startsWith('/metrics/historical/') || req.path.startsWith('/metrics/data')) {
        console.log('\n=== Processing Historical Metrics Request ===');

        let resourceArn;
        if (req.path.startsWith('/metrics/historical/')) {
            const instanceId = req.path.split('/metrics/historical/')[1]?.split('?')[0];
            console.log('📍 Extracted instanceId from historical path:', instanceId);
            resourceArn = `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${instanceId}`;
        } else if (req.path.startsWith('/metrics/data')) {
            // Extract from request body since /metrics/data is typically POST
            resourceArn = req.body.ResourceArn;
            console.log('📍 Using resourceArn from body for /metrics/data:', resourceArn);
        } else {
            resourceArn = req.body.ResourceArn;
            console.log('📍 Using resourceArn from body:', resourceArn);
        }

        if (!resourceArn || !resourceArn.includes(INSTANCE_ID)) {
            console.log('❌ Invalid resourceArn:', resourceArn);
            return res.status(404).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
        }

        try {
            const body = req.body;
            console.log('\n📦 Request body:', JSON.stringify(body, null, 2));

            if (!body.Filters || !body.Metrics) {
                console.log('❌ Missing Filters or Metrics field');
                return res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
            }

            console.log('\n🔍 Validating metrics...');
            const metricsValid = validateMetrics(body.Metrics);
            if (!metricsValid) {
                console.log('❌ Metric validation failed');
                return res.status(400).json(AWS_ERRORS.VALIDATION_EXCEPTION);
            }

            console.log('\n🔍 Validating filters...');
            const filtersValid = validateFiltersV2(body.Filters);
            if (!filtersValid) {
                console.log('❌ Filter validation failed');
                return res.status(400).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
            }

            const error = shouldError();
            if (error) {
                console.log('⚠️ Random error triggered:', error);
                return res.status(500).json(error);
            }

            const allMetricResults = [];
            
            // Extract queue IDs from FilterV2 structure
            const queueFilter = body.Filters.find(f => f.FilterKey === "QUEUE");
            const queueIds = queueFilter ? queueFilter.FilterValues : [];

            // Extract channel filters from FilterV2 structure
            const channelFilter = body.Filters.find(f => f.FilterKey === "CHANNEL");
            const channelFilters = channelFilter ? channelFilter.FilterValues : [];

            queueIds.forEach(queueId => {
                const queue = Object.values(MOCK_QUEUES).find(q => q.Id === queueId);
                if (queue) {
                    if (channelFilters.length > 0 && !channelFilters.includes(queue.Channel)) {
                        console.log(`Skipping queue ${queue.Id} due to channel filter`);
                        return;
                    }
                    console.log(`Generating historical metrics for queue: ${queue.Name} (${queue.Id})`);
                    const metrics = generateMetricsV2(
                        queue,
                        body.Metrics
                    );
                    console.log('Generated metrics:', JSON.stringify(metrics, null, 2));
                    allMetricResults.push(...metrics);
                }
            });

            const { items, nextToken: newNextToken } = paginateResults(
                allMetricResults,
                body.NextToken
            );

            const response = {
                MetricResults: items,
                NextToken: newNextToken,
                DataSnapshotTime: Date.now()
            };

            console.log('\n✅ Sending response:', JSON.stringify(response, null, 2));
            return res.json(response);

        } catch (error) {
            console.error('\n❌ Internal error:', error);
            return res.status(500).json(AWS_ERRORS.INTERNAL_SERVICE_ERROR);
        }
    }
    else if (action === 'GetCurrentUserData' || req.path.startsWith('/current-user-data/') || req.path.startsWith('/metrics/userdata/')) {
        console.log('\n=== Processing Current User Data Request ===');

        let instanceId;
        if (req.path.startsWith('/current-user-data/')) {
            instanceId = req.path.split('/current-user-data/')[1]?.split('?')[0];
            console.log('📍 Extracted instanceId from path:', instanceId);
        } else if (req.path.startsWith('/metrics/userdata/')) {
            instanceId = req.path.split('/metrics/userdata/')[1]?.split('?')[0];
            console.log('📍 Extracted instanceId from userdata path:', instanceId);
        } else {
            instanceId = req.body.InstanceId || req.query.InstanceId;
            console.log('📍 Using instanceId from request:', instanceId);
        }

        if (!instanceId || instanceId !== INSTANCE_ID) {
            console.log('❌ Invalid instanceId:', instanceId);
            return res.status(404).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
        }

        const nextToken = req.query.nextToken || (req.body && req.body.NextToken);
        if (nextToken && isNaN(parseInt(nextToken, 10))) {
            console.log('Invalid nextToken:', nextToken);
            return res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
        }

        const error = shouldError();
        if (error) {
            console.log('Random error triggered:', error);
            return res.status(500).json(error);
        }

        const { items, nextToken: newNextToken } = paginateResults(MOCK_USERS, nextToken);
        const formatted = items.map(u => ({
            User: { Id: u.Id, Arn: u.Arn, Username: u.Username },
            Status: u.Status,
            RoutingProfile: u.RoutingProfile
        }));
        return res.json({
            UserDataList: formatted,
            NextToken: newNextToken,
            ApproximateTotalCount: MOCK_USERS.length
        });
    }
    else if (action === 'ListRoutingProfileQueues' || req.path.startsWith('/routing-profile-queues/') || req.path.includes('/routing-profiles/')) {
        console.log('\n=== Processing Routing Profile Queues Request ===');

        let instanceId, rpId;
        if (req.path.startsWith('/routing-profile-queues/')) {
            const parts = req.path.split('/routing-profile-queues/')[1]?.split('/') || [];
            instanceId = parts[0];
            rpId = parts[1]?.split('?')[0];
            console.log('📍 Extracted instanceId and rpId from path:', instanceId, rpId);
        } else if (req.path.includes('/routing-profiles/')) {
            // Handle AWS SDK URL pattern: /routing-profiles/{instanceId}/{routingProfileId}/queues
            const match = req.path.match(/\/routing-profiles\/([^\/]+)\/([^\/]+)\/queues/);
            if (match) {
                instanceId = match[1];
                rpId = match[2];
                console.log('📍 Extracted instanceId and rpId from AWS SDK path:', instanceId, rpId);
            }
        } else {
            instanceId = req.body.InstanceId || req.query.InstanceId;
            rpId = req.body.RoutingProfileId || req.query.RoutingProfileId;
            console.log('📍 Using instanceId/rpId from request:', instanceId, rpId);
        }

        if (!instanceId || instanceId !== INSTANCE_ID) {
            console.log('❌ Invalid instanceId:', instanceId);
            return res.status(404).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
        }
        if (!rpId || !ROUTING_PROFILES[rpId]) {
            console.log('❌ Invalid routingProfileId:', rpId);
            return res.status(404).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
        }

        const nextToken = req.query.nextToken || (req.body && req.body.NextToken);
        if (nextToken && isNaN(parseInt(nextToken, 10))) {
            console.log('Invalid nextToken:', nextToken);
            return res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
        }

        const error = shouldError();
        if (error) {
            console.log('Random error triggered:', error);
            return res.status(500).json(error);
        }

        const queues = ROUTING_PROFILES[rpId].Queues.map(qid => {
            const q = Object.values(MOCK_QUEUES).find(q => q.Id === qid);
            if (!q) return null;
            return {
                QueueArn: q.Arn,
                QueueId: q.Id,
                QueueName: q.Name,
                Channel: q.Channel,
                Priority: q.Priority,
                Delay: 0
            };
        }).filter(Boolean);

        const { items, nextToken: newNextToken } = paginateResults(queues, nextToken);
        return res.json({
            RoutingProfileQueueConfigSummaryList: items,
            NextToken: newNextToken
        });
    }
    else if (action === 'ListAgentStatuses' || req.path.startsWith('/agent-status/')) {
        console.log('\n=== Processing Agent Status Request ===');

        let instanceId;
        if (req.path.startsWith('/agent-status/')) {
            instanceId = req.path.split('/agent-status/')[1]?.split('?')[0];
            console.log('📍 Extracted instanceId from path:', instanceId);
        } else {
            instanceId = req.query.InstanceId || req.body.InstanceId;
            console.log('📍 Using instanceId from request:', instanceId);
        }

        if (!instanceId || instanceId !== INSTANCE_ID) {
            console.log('❌ Invalid instanceId:', instanceId);
            return res.status(404).json(AWS_ERRORS.RESOURCE_NOT_FOUND);
        }

        const nextToken = req.query.nextToken || (req.body && req.body.NextToken);
        if (nextToken && isNaN(parseInt(nextToken, 10))) {
            console.log('Invalid nextToken:', nextToken);
            return res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
        }

        const error = shouldError();
        if (error) {
            console.log('Random error triggered:', error);
            return res.status(500).json(error);
        }

        // Return static status definitions (not agent-specific data)
        // Map the predefined agent statuses to match the ARNs used in MOCK_USERS
        const statusArns = {
            'available-status-id': 'status-001',
            'acw-status-id': 'status-003', 
            'oncall-status-id': 'status-002',
            'lunch-status-id': 'status-004',
            'error-status-id': 'status-005'
        };
        
        const statusList = agentStatuses.map(status => {
            const statusArn = statusArns[status.Id] || status.Id;
            return {
                Arn: `arn:aws:connect:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}/agent-state/${statusArn}`,
                Id: status.Id,
                Name: status.Name,
                Type: status.Type
            };
        });

        // Respect MaxResults from the request (AWS Connect collectors often set this to 1000)
        const maxResults = (req.body && req.body.MaxResults) || req.query.MaxResults || ITEMS_PER_PAGE;
        console.log(`Using MaxResults: ${maxResults} for ListAgentStatuses`);
        
        const startIndex = nextToken ? parseInt(nextToken, 10) : 0;
        const endIndex = Math.min(startIndex + maxResults, statusList.length);
        const items = statusList.slice(startIndex, endIndex);
        const newNextToken = endIndex < statusList.length ? endIndex.toString() : null;
        
        return res.json({
            AgentStatusSummaryList: items,
            NextToken: newNextToken
        });
    }
    
    // If not an AWS SDK request, continue to other routes
    next();
});
/**
 * POST /admin/errorRate
 * 
 * Allows an admin (or a developer) to set a new error rate 
 * at runtime. Must be a number between 0 and 1, inclusive.
 * Example body: { "rate": 0.5 }
 */
app.post('/admin/errorRate', (req, res) => {
  // Check if the "rate" field is a valid number
  if (typeof req.body.rate === 'number') {
    // Clamp the value between 0 and 1
    errorRate = Math.max(0, Math.min(1, req.body.rate));
    res.json({ message: `Error rate set to ${errorRate * 100}%` });
  } else {
    // If it's not a valid number, respond with an INVALID_PARAMETER error
    res.status(400).json(AWS_ERRORS.INVALID_PARAMETER);
  }
});

// Optional admin endpoint to reset historical totals for easier testing
app.post('/admin/resetTotals', (req, res) => {
  Object.keys(queueTotals).forEach(qid => {
    Object.keys(queueTotals[qid]).forEach(metric => {
      queueTotals[qid][metric] = 0;
    });
  });
  res.json({ message: 'Queue totals reset' });
});

// Basic liveness check
app.get('/health', (req, res) => {
    res.status(200).send('OK');
});

//OPTIONS handler for CORS preflight requests
// Express 5.x also rejects `*` here, so use a wildcard parameter
app.options('/{*path}', (req, res) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Methods', 'GET,PUT,POST,DELETE,OPTIONS');
    res.header('Access-Control-Allow-Headers', '*');
    res.sendStatus(200);
});

// Handle unmatched routes
app.use((req, res) => {
    console.log('Unmatched route:', req.path);
    res.status(404).json({
        __type: 'UnknownOperationException',
        message: `Unknown operation: ${req.path}`
    });
});

/****************************************************
 * SERVER START
 ****************************************************/

// Start the Express server on port 3000 or a custom port if set in the environment
const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Mock AWS Connect API running on port ${PORT}`);
  console.log(`Instance ID: ${INSTANCE_ID}`);
  console.log(`Error rate: ${errorRate * 100}%`);
});
