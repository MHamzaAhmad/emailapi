/**
 * HTTP Smoke Test - Baseline Performance
 * 
 * Purpose: Measure "clean" latency with minimal load
 * Config: 10 RPS for 30 seconds
 * Goal: Establish your performance "floor"
 * 
 * Set DRY_RUN=true to test without sending actual emails (saves SES quota)
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

// Custom metrics
const emailLatency = new Trend('email_send_latency', true);
const successRate = new Rate('success_rate');
const emailsSent = new Counter('emails_sent');

// Configuration from environment
const API_KEY = __ENV.API_KEY;
const API_HOST = __ENV.API_HOST || 'api.simpleemailapi.dev';
const API_PORT = __ENV.API_PORT || '443';
const FROM_EMAIL = __ENV.FROM_EMAIL;
const TO_EMAIL = __ENV.TO_EMAIL;
const DRY_RUN = __ENV.DRY_RUN === 'true';

const BASE_URL = API_PORT === '443'
    ? `https://${API_HOST}`
    : `http://${API_HOST}:${API_PORT}`;

export const options = {
    scenarios: {
        smoke: {
            executor: 'constant-arrival-rate',
            rate: 10,              // 10 RPS
            timeUnit: '1s',
            duration: '30s',
            preAllocatedVUs: 20,
            maxVUs: 50,
        },
    },
    thresholds: {
        'email_send_latency': ['p(50)<100', 'p(95)<300', 'p(99)<500'],
        'success_rate': ['rate>0.99'],
        'http_req_failed': ['rate<0.01'],
    },
};

export default function () {
    const timestamp = Date.now();

    const payload = JSON.stringify({
        from: FROM_EMAIL,
        to: [TO_EMAIL],
        subject: `HTTP Smoke Test ${timestamp}`,
        body: `Performance smoke test at ${new Date().toISOString()}`,
    });

    const headers = {
        'Authorization': `Bearer ${API_KEY}`,
        'Content-Type': 'application/json',
    };

    // Add dry-run header to skip actually sending emails (saves SES quota)
    if (DRY_RUN) {
        headers['X-Dry-Run'] = 'true';
    }

    const params = {
        headers: headers,
        timeout: '10s',
    };

    const startTime = Date.now();
    const res = http.post(`${BASE_URL}/v1.EmailService/SendEmail`, payload, params);
    const duration = Date.now() - startTime;

    // Record custom metrics
    emailLatency.add(duration);
    emailsSent.add(1);

    const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'has email id': (r) => {
            try {
                const body = JSON.parse(r.body);
                return body.id !== undefined;
            } catch {
                return false;
            }
        },
        'latency < 500ms': () => duration < 500,
    });

    successRate.add(success);

    if (!success) {
        console.log(`Failed request: status=${res.status}, body=${res.body}`);
    }
}

export function handleSummary(data) {
    const summary = {
        test_type: 'smoke',
        protocol: 'http',
        timestamp: new Date().toISOString(),
        config: {
            rps: 10,
            duration: '30s',
            host: API_HOST,
        },
        results: {
            total_requests: data.metrics.http_reqs?.values?.count || 0,
            success_rate: data.metrics.success_rate?.values?.rate || 0,
            latency: {
                min: data.metrics.email_send_latency?.values?.min || 0,
                avg: data.metrics.email_send_latency?.values?.avg || 0,
                med: data.metrics.email_send_latency?.values?.med || 0,
                p95: data.metrics.email_send_latency?.values?.['p(95)'] || 0,
                p99: data.metrics.email_send_latency?.values?.['p(99)'] || 0,
                max: data.metrics.email_send_latency?.values?.max || 0,
            },
            errors: data.metrics.http_req_failed?.values?.rate || 0,
        },
    };

    console.log('\n' + '═'.repeat(60));
    console.log('  SMOKE TEST SUMMARY');
    console.log('═'.repeat(60));
    console.log(`  Total Requests: ${summary.results.total_requests}`);
    console.log(`  Success Rate:   ${(summary.results.success_rate * 100).toFixed(2)}%`);
    console.log('');
    console.log('  Latency Distribution:');
    console.log(`    Min:  ${summary.results.latency.min.toFixed(2)}ms`);
    console.log(`    P50:  ${summary.results.latency.med.toFixed(2)}ms`);
    console.log(`    P95:  ${summary.results.latency.p95.toFixed(2)}ms`);
    console.log(`    P99:  ${summary.results.latency.p99.toFixed(2)}ms`);
    console.log(`    Max:  ${summary.results.latency.max.toFixed(2)}ms`);
    console.log('═'.repeat(60) + '\n');

    return {
        stdout: JSON.stringify(summary, null, 2),
    };
}
