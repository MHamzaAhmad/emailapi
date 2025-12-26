/**
 * HTTP Stress Test - Find Breaking Point
 * 
 * Purpose: Push the API until it starts failing
 * Config: Ramp from 50 to 500 RPS over 5 minutes
 * Goal: Find your "Max Capacity" - when do errors start appearing?
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
const errorsByStatus = new Counter('errors_by_status');

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
        stress: {
            executor: 'ramping-arrival-rate',
            startRate: 50,
            timeUnit: '1s',
            preAllocatedVUs: 500,
            maxVUs: 1000,
            stages: [
                { duration: '1m', target: 100 },   // Ramp to 100 RPS
                { duration: '1m', target: 200 },   // Ramp to 200 RPS
                { duration: '1m', target: 300 },   // Ramp to 300 RPS
                { duration: '1m', target: 400 },   // Ramp to 400 RPS
                { duration: '1m', target: 500 },   // Ramp to 500 RPS
                { duration: '30s', target: 0 },    // Ramp down
            ],
        },
    },
    thresholds: {
        // Relaxed thresholds for stress test - we expect some failures
        'success_rate': ['rate>0.90'],        // Allow up to 10% failure
        'http_req_failed': ['rate<0.10'],
    },
};

// Track errors over time for analysis
let errorLog = [];

export default function () {
    const timestamp = Date.now();

    const payload = JSON.stringify({
        from: FROM_EMAIL,
        to: [TO_EMAIL],
        subject: `HTTP Stress Test ${timestamp}`,
        body: `Performance stress test at ${new Date().toISOString()}`,
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
        timeout: '30s',  // Longer timeout for stress test
    };

    const startTime = Date.now();
    const res = http.post(`${BASE_URL}/v1.EmailService/SendEmail`, payload, params);
    const duration = Date.now() - startTime;

    // Record custom metrics
    emailLatency.add(duration);
    emailsSent.add(1);

    const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'latency < 2s': () => duration < 2000,
    });

    successRate.add(success);

    if (!success) {
        errorsByStatus.add(1, { status: res.status.toString() });

        // Log first few errors for debugging
        if (errorLog.length < 50) {
            errorLog.push({
                timestamp: Date.now(),
                status: res.status,
                duration: duration,
                body: res.body?.substring(0, 100),
            });
        }
    }
}

export function handleSummary(data) {
    const totalRequests = data.metrics.http_reqs?.values?.count || 0;
    const errorRate = data.metrics.http_req_failed?.values?.rate || 0;
    const successRateVal = data.metrics.success_rate?.values?.rate || 0;

    const summary = {
        test_type: 'stress',
        protocol: 'http',
        timestamp: new Date().toISOString(),
        config: {
            start_rps: 50,
            end_rps: 500,
            duration: '5m30s',
            host: API_HOST,
        },
        results: {
            total_requests: totalRequests,
            success_rate: successRateVal,
            error_rate: errorRate,
            latency: {
                min: data.metrics.email_send_latency?.values?.min || 0,
                avg: data.metrics.email_send_latency?.values?.avg || 0,
                med: data.metrics.email_send_latency?.values?.med || 0,
                p95: data.metrics.email_send_latency?.values?.['p(95)'] || 0,
                p99: data.metrics.email_send_latency?.values?.['p(99)'] || 0,
                max: data.metrics.email_send_latency?.values?.max || 0,
            },
            estimated_max_rps: calculateMaxRPS(totalRequests, successRateVal),
        },
    };

    console.log('\n' + '═'.repeat(60));
    console.log('  STRESS TEST SUMMARY - FINDING BREAKING POINT');
    console.log('═'.repeat(60));
    console.log(`  Total Requests: ${summary.results.total_requests}`);
    console.log(`  Success Rate:   ${(summary.results.success_rate * 100).toFixed(2)}%`);
    console.log(`  Error Rate:     ${(summary.results.error_rate * 100).toFixed(2)}%`);
    console.log('');
    console.log('  Latency Distribution:');
    console.log(`    P50:  ${summary.results.latency.med.toFixed(2)}ms`);
    console.log(`    P95:  ${summary.results.latency.p95.toFixed(2)}ms`);
    console.log(`    P99:  ${summary.results.latency.p99.toFixed(2)}ms`);
    console.log(`    Max:  ${summary.results.latency.max.toFixed(2)}ms`);
    console.log('');
    console.log('  Capacity Estimate:');
    console.log(`    Estimated Max Sustainable RPS: ~${summary.results.estimated_max_rps}`);
    console.log('');

    if (errorRate > 0.05) {
        console.log('  ⚠️  BREAKING POINT REACHED');
        console.log('     - Consider scaling up your infrastructure');
        console.log('     - Check NGINX connection limits');
        console.log('     - Review database connection pool size');
    } else if (errorRate > 0.01) {
        console.log('  ⚡ Minor errors detected');
        console.log('     - You are approaching capacity limits');
    } else {
        console.log('  ✅ API handled stress test well!');
        console.log('     - Consider testing with higher RPS');
    }

    console.log('═'.repeat(60) + '\n');

    return {
        stdout: JSON.stringify(summary, null, 2),
    };
}

function calculateMaxRPS(totalRequests, successRate) {
    // Rough estimate based on achieved throughput and success rate
    const testDuration = 330; // 5m30s in seconds
    const avgRPS = totalRequests / testDuration;

    // If success rate drops significantly, max sustainable RPS is lower
    if (successRate < 0.95) {
        // Estimate where we'd hit 99% success rate
        return Math.floor(avgRPS * 0.7);
    } else if (successRate < 0.99) {
        return Math.floor(avgRPS * 0.9);
    }

    // If we maintained >99% success, we can likely handle more
    return Math.floor(avgRPS * 1.1);
}
