/**
 * HTTP Load Test - Realistic Production Traffic
 * 
 * Purpose: Simulate typical production traffic
 * Config: 100 RPS sustained for 2 minutes
 * Goal: Does P99 stay under 500ms? If P99 spikes while P50 stays flat,
 *       you have a concurrency bottleneck (likely DB or Redis connections).
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
        load: {
            executor: 'constant-arrival-rate',
            rate: 100,             // 100 RPS
            timeUnit: '1s',
            duration: '2m',
            preAllocatedVUs: 150,
            maxVUs: 300,
        },
    },
    thresholds: {
        'email_send_latency': ['p(50)<150', 'p(95)<400', 'p(99)<800'],
        'success_rate': ['rate>0.98'],
        'http_req_failed': ['rate<0.02'],
    },
};

export default function () {
    const timestamp = Date.now();

    const payload = JSON.stringify({
        from: FROM_EMAIL,
        to: [TO_EMAIL],
        subject: `HTTP Load Test ${timestamp}`,
        body: `Performance load test at ${new Date().toISOString()}`,
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
        timeout: '15s',
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
        'latency < 1s': () => duration < 1000,
    });

    successRate.add(success);

    if (!success && res.status !== 200) {
        console.log(`Failed request: status=${res.status}, body=${res.body?.substring(0, 200)}`);
    }
}

export function handleSummary(data) {
    const summary = {
        test_type: 'load',
        protocol: 'http',
        timestamp: new Date().toISOString(),
        config: {
            rps: 100,
            duration: '2m',
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

    // Calculate if there's a tail latency issue
    const p50 = summary.results.latency.med;
    const p99 = summary.results.latency.p99;
    const tailRatio = p99 / p50;

    console.log('\n' + '═'.repeat(60));
    console.log('  LOAD TEST SUMMARY');
    console.log('═'.repeat(60));
    console.log(`  Total Requests: ${summary.results.total_requests}`);
    console.log(`  Success Rate:   ${(summary.results.success_rate * 100).toFixed(2)}%`);
    console.log(`  Throughput:     ~${(summary.results.total_requests / 120).toFixed(1)} req/s`);
    console.log('');
    console.log('  Latency Distribution:');
    console.log(`    Min:  ${summary.results.latency.min.toFixed(2)}ms`);
    console.log(`    P50:  ${summary.results.latency.med.toFixed(2)}ms`);
    console.log(`    P95:  ${summary.results.latency.p95.toFixed(2)}ms`);
    console.log(`    P99:  ${summary.results.latency.p99.toFixed(2)}ms`);
    console.log(`    Max:  ${summary.results.latency.max.toFixed(2)}ms`);
    console.log('');
    console.log('  Analysis:');
    console.log(`    P99/P50 Ratio: ${tailRatio.toFixed(2)}x`);
    if (tailRatio > 5) {
        console.log('    ⚠️  HIGH TAIL LATENCY - Check for connection pool exhaustion');
    } else if (tailRatio > 3) {
        console.log('    ⚡ Moderate tail latency - Consider investigating slow queries');
    } else {
        console.log('    ✅ Tail latency is healthy');
    }
    console.log('═'.repeat(60) + '\n');

    return {
        stdout: JSON.stringify(summary, null, 2),
    };
}
