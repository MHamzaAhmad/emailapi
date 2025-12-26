#!/bin/bash

# =============================================================================
# Full Performance Benchmark Suite
# 
# Runs a complete benchmark across all modes and generates a consolidated report.
# Usage: ./benchmark.sh [--dry-run] [--quick]
#
# Options:
#   --dry-run    Test without sending actual emails (recommended for SES sandbox)
#   --quick      Run quick tests only (5s each instead of 30s)
#   --http-only  Only test HTTP endpoint
#   --grpc-only  Only test gRPC endpoint
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"
PROTO_DIR="$SCRIPT_DIR/../../proto"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
REPORT_FILE="$RESULTS_DIR/benchmark-report-$TIMESTAMP.md"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Options
DRY_RUN=""
QUICK_MODE=""
HTTP_ONLY=""
GRPC_ONLY=""

# Results storage
declare -A RESULTS

# =============================================================================
# Parse Arguments
# =============================================================================
for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN="true"
            ;;
        --quick)
            QUICK_MODE="true"
            ;;
        --http-only)
            HTTP_ONLY="true"
            ;;
        --grpc-only)
            GRPC_ONLY="true"
            ;;
        --help|-h)
            echo "Usage: $0 [--dry-run] [--quick] [--http-only] [--grpc-only]"
            echo ""
            echo "Options:"
            echo "  --dry-run    Test without sending actual emails"
            echo "  --quick      Run quick tests (5s each instead of 30s)"
            echo "  --http-only  Only test HTTP endpoint"
            echo "  --grpc-only  Only test gRPC endpoint"
            exit 0
            ;;
    esac
done

# =============================================================================
# Configuration
# =============================================================================
load_config() {
    if [[ -f "$SCRIPT_DIR/.env" ]]; then
        export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
    fi

    if [[ -z "$PERF_API_KEY" || -z "$PERF_FROM_EMAIL" || -z "$PERF_TO_EMAIL" ]]; then
        echo -e "${RED}ERROR: Missing required environment variables${NC}"
        echo "Required: PERF_API_KEY, PERF_FROM_EMAIL, PERF_TO_EMAIL"
        echo "Set them in tools/perf/.env"
        exit 1
    fi

    PERF_API_HOST="${PERF_API_HOST:-api.simpleemailapi.dev}"
    PERF_GRPC_PORT="${PERF_GRPC_PORT:-443}"
    PERF_HTTP_PORT="${PERF_HTTP_PORT:-443}"
}

# =============================================================================
# Utility Functions
# =============================================================================
check_tools() {
    local missing=""
    
    if [[ -z "$HTTP_ONLY" ]] && ! command -v ghz &> /dev/null; then
        missing="$missing ghz"
    fi
    
    if [[ -z "$GRPC_ONLY" ]] && ! command -v k6 &> /dev/null; then
        missing="$missing k6"
    fi
    
    if [[ -n "$missing" ]]; then
        echo -e "${RED}ERROR: Missing tools:$missing${NC}"
        echo "Install with: brew install ghz k6"
        exit 1
    fi
}

print_banner() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}${BOLD}        SimpleEmailAPI Performance Benchmark Suite        ${NC}${BLUE}║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${YELLOW}Configuration:${NC}"
    echo "  Host:       $PERF_API_HOST"
    echo "  From:       $PERF_FROM_EMAIL"
    echo "  To:         $PERF_TO_EMAIL"
    if [[ -n "$DRY_RUN" ]]; then
        echo -e "  Mode:       ${CYAN}DRY-RUN (no emails sent, ~50-150ms simulated SES delay)${NC}"
    else
        echo -e "  Mode:       ${GREEN}LIVE (actual emails will be sent)${NC}"
    fi
    if [[ -n "$QUICK_MODE" ]]; then
        echo -e "  Duration:   ${CYAN}QUICK (5s per test)${NC}"
    else
        echo "  Duration:   Standard (15-30s per test)"
    fi
    echo ""
}

# =============================================================================
# HTTP Tests with k6
# =============================================================================
run_http_sync_test() {
    local name=$1
    local rps=$2
    local duration=$3
    
    echo -e "${CYAN}► HTTP Sync Mode @ ${rps} RPS (${duration})${NC}"
    
    local output_file="$RESULTS_DIR/http-sync-${rps}rps-$TIMESTAMP.json"
    local dry_run_env=""
    [[ -n "$DRY_RUN" ]] && dry_run_env="--env DRY_RUN=true"
    
    # Create inline k6 script for sync mode
    k6 run --quiet \
        --env API_KEY="$PERF_API_KEY" \
        --env API_HOST="$PERF_API_HOST" \
        --env API_PORT="$PERF_HTTP_PORT" \
        --env FROM_EMAIL="$PERF_FROM_EMAIL" \
        --env TO_EMAIL="$PERF_TO_EMAIL" \
        --env ASYNC_MODE="false" \
        --env TARGET_RPS="$rps" \
        --env DURATION="$duration" \
        $dry_run_env \
        --summary-export="$output_file" \
        - <<'SCRIPT'
import http from 'k6/http';
import { check } from 'k6';
import { Trend } from 'k6/metrics';

const latency = new Trend('email_latency', true);
const API_KEY = __ENV.API_KEY;
const API_HOST = __ENV.API_HOST || 'api.simpleemailapi.dev';
const API_PORT = __ENV.API_PORT || '443';
const FROM_EMAIL = __ENV.FROM_EMAIL;
const TO_EMAIL = __ENV.TO_EMAIL;
const DRY_RUN = __ENV.DRY_RUN === 'true';
const ASYNC_MODE = __ENV.ASYNC_MODE === 'true';
const TARGET_RPS = parseInt(__ENV.TARGET_RPS) || 10;
const DURATION = __ENV.DURATION || '15s';

const BASE_URL = API_PORT === '443' ? `https://${API_HOST}` : `http://${API_HOST}:${API_PORT}`;

export const options = {
    scenarios: {
        benchmark: {
            executor: 'constant-arrival-rate',
            rate: TARGET_RPS,
            timeUnit: '1s',
            duration: DURATION,
            preAllocatedVUs: Math.max(50, TARGET_RPS * 2),
            maxVUs: Math.max(100, TARGET_RPS * 3),
        },
    },
};

export default function () {
    const headers = {
        'Authorization': `Bearer ${API_KEY}`,
        'Content-Type': 'application/json',
    };
    if (DRY_RUN) headers['X-Dry-Run'] = 'true';

    const payload = JSON.stringify({
        from: FROM_EMAIL,
        to: [TO_EMAIL],
        subject: `Benchmark ${Date.now()}`,
        body: 'Performance benchmark test',
        async: ASYNC_MODE,
    });

    const start = Date.now();
    const res = http.post(`${BASE_URL}/v1/email`, payload, { headers, timeout: '30s' });
    latency.add(Date.now() - start);

    check(res, { 'status is 200': (r) => r.status === 200 });
}
SCRIPT

    # Parse results
    if [[ -f "$output_file" ]]; then
        local p50=$(jq -r '.metrics.email_latency.values.med // .metrics.http_req_duration.values.med // 0' "$output_file")
        local p95=$(jq -r '.metrics.email_latency.values["p(95)"] // .metrics.http_req_duration.values["p(95)"] // 0' "$output_file")
        local p99=$(jq -r '.metrics.email_latency.values["p(99)"] // .metrics.http_req_duration.values["p(99)"] // 0' "$output_file")
        local success=$(jq -r '.metrics.checks.values.rate // 1' "$output_file")
        
        RESULTS["http_sync_${rps}_p50"]="${p50%.*}"
        RESULTS["http_sync_${rps}_p95"]="${p95%.*}"
        RESULTS["http_sync_${rps}_p99"]="${p99%.*}"
        RESULTS["http_sync_${rps}_success"]=$(echo "$success * 100" | bc | cut -d'.' -f1)
        
        echo -e "  P50: ${p50%.*}ms | P95: ${p95%.*}ms | P99: ${p99%.*}ms | Success: $(echo "$success * 100" | bc | cut -d'.' -f1)%"
    fi
}

run_http_async_test() {
    local name=$1
    local rps=$2
    local duration=$3
    
    echo -e "${CYAN}► HTTP Async Mode @ ${rps} RPS (${duration})${NC}"
    
    local output_file="$RESULTS_DIR/http-async-${rps}rps-$TIMESTAMP.json"
    local dry_run_env=""
    [[ -n "$DRY_RUN" ]] && dry_run_env="--env DRY_RUN=true"
    
    k6 run --quiet \
        --env API_KEY="$PERF_API_KEY" \
        --env API_HOST="$PERF_API_HOST" \
        --env API_PORT="$PERF_HTTP_PORT" \
        --env FROM_EMAIL="$PERF_FROM_EMAIL" \
        --env TO_EMAIL="$PERF_TO_EMAIL" \
        --env ASYNC_MODE="true" \
        --env TARGET_RPS="$rps" \
        --env DURATION="$duration" \
        $dry_run_env \
        --summary-export="$output_file" \
        - <<'SCRIPT'
import http from 'k6/http';
import { check } from 'k6';
import { Trend } from 'k6/metrics';

const latency = new Trend('email_latency', true);
const API_KEY = __ENV.API_KEY;
const API_HOST = __ENV.API_HOST || 'api.simpleemailapi.dev';
const API_PORT = __ENV.API_PORT || '443';
const FROM_EMAIL = __ENV.FROM_EMAIL;
const TO_EMAIL = __ENV.TO_EMAIL;
const DRY_RUN = __ENV.DRY_RUN === 'true';
const ASYNC_MODE = __ENV.ASYNC_MODE === 'true';
const TARGET_RPS = parseInt(__ENV.TARGET_RPS) || 10;
const DURATION = __ENV.DURATION || '15s';

const BASE_URL = API_PORT === '443' ? `https://${API_HOST}` : `http://${API_HOST}:${API_PORT}`;

export const options = {
    scenarios: {
        benchmark: {
            executor: 'constant-arrival-rate',
            rate: TARGET_RPS,
            timeUnit: '1s',
            duration: DURATION,
            preAllocatedVUs: Math.max(50, TARGET_RPS * 2),
            maxVUs: Math.max(100, TARGET_RPS * 3),
        },
    },
};

export default function () {
    const headers = {
        'Authorization': `Bearer ${API_KEY}`,
        'Content-Type': 'application/json',
    };
    if (DRY_RUN) headers['X-Dry-Run'] = 'true';

    const payload = JSON.stringify({
        from: FROM_EMAIL,
        to: [TO_EMAIL],
        subject: `Benchmark ${Date.now()}`,
        body: 'Performance benchmark test',
        async: ASYNC_MODE,
    });

    const start = Date.now();
    const res = http.post(`${BASE_URL}/v1/email`, payload, { headers, timeout: '30s' });
    latency.add(Date.now() - start);

    check(res, { 'status is 200': (r) => r.status === 200 });
}
SCRIPT

    if [[ -f "$output_file" ]]; then
        local p50=$(jq -r '.metrics.email_latency.values.med // .metrics.http_req_duration.values.med // 0' "$output_file")
        local p95=$(jq -r '.metrics.email_latency.values["p(95)"] // .metrics.http_req_duration.values["p(95)"] // 0' "$output_file")
        local p99=$(jq -r '.metrics.email_latency.values["p(99)"] // .metrics.http_req_duration.values["p(99)"] // 0' "$output_file")
        local success=$(jq -r '.metrics.checks.values.rate // 1' "$output_file")
        
        RESULTS["http_async_${rps}_p50"]="${p50%.*}"
        RESULTS["http_async_${rps}_p95"]="${p95%.*}"
        RESULTS["http_async_${rps}_p99"]="${p99%.*}"
        RESULTS["http_async_${rps}_success"]=$(echo "$success * 100" | bc | cut -d'.' -f1)
        
        echo -e "  P50: ${p50%.*}ms | P95: ${p95%.*}ms | P99: ${p99%.*}ms | Success: $(echo "$success * 100" | bc | cut -d'.' -f1)%"
    fi
}

# =============================================================================
# gRPC Tests with ghz
# =============================================================================
run_grpc_test() {
    local mode=$1  # sync or async
    local rps=$2
    local duration=$3
    
    echo -e "${CYAN}► gRPC ${mode^} Mode @ ${rps} RPS (${duration})${NC}"
    
    local output_file="$RESULTS_DIR/grpc-${mode}-${rps}rps-$TIMESTAMP.json"
    
    local metadata
    if [[ -n "$DRY_RUN" ]]; then
        metadata="{\"authorization\": \"Bearer $PERF_API_KEY\", \"x-dry-run\": \"true\"}"
    else
        metadata="{\"authorization\": \"Bearer $PERF_API_KEY\"}"
    fi
    
    local async_flag="false"
    [[ "$mode" == "async" ]] && async_flag="true"
    
    ghz --insecure="${GRPC_INSECURE:-false}" \
        --proto="$PROTO_DIR/public/v1/email.proto" \
        --import-paths="$PROTO_DIR/public" \
        --call=v1.EmailService/SendEmail \
        --rps="$rps" \
        --duration="$duration" \
        --connections=20 \
        --concurrency=$((rps * 2)) \
        --metadata="$metadata" \
        --data="{\"from\": \"$PERF_FROM_EMAIL\", \"to\": [\"$PERF_TO_EMAIL\"], \"subject\": \"gRPC Benchmark $(date +%s)\", \"body\": \"Performance test\", \"async\": $async_flag}" \
        --format=json \
        --output="$output_file" \
        "$PERF_API_HOST:$PERF_GRPC_PORT" 2>/dev/null
    
    if [[ -f "$output_file" ]]; then
        local p50=$(jq -r '.latencyDistribution[] | select(.percentage == 50) | .latency' "$output_file" 2>/dev/null | sed 's/ms//' || echo "0")
        local p95=$(jq -r '.latencyDistribution[] | select(.percentage == 95) | .latency' "$output_file" 2>/dev/null | sed 's/ms//' || echo "0")
        local p99=$(jq -r '.latencyDistribution[] | select(.percentage == 99) | .latency' "$output_file" 2>/dev/null | sed 's/ms//' || echo "0")
        local total=$(jq -r '.count' "$output_file" 2>/dev/null || echo "0")
        local errors=$(jq -r '.errorCount // 0' "$output_file" 2>/dev/null || echo "0")
        
        if [[ "$total" -gt 0 ]]; then
            local success=$(echo "scale=0; (($total - $errors) * 100) / $total" | bc)
        else
            local success=0
        fi
        
        RESULTS["grpc_${mode}_${rps}_p50"]="${p50%.*}"
        RESULTS["grpc_${mode}_${rps}_p95"]="${p95%.*}"
        RESULTS["grpc_${mode}_${rps}_p99"]="${p99%.*}"
        RESULTS["grpc_${mode}_${rps}_success"]="$success"
        
        echo -e "  P50: ${p50}ms | P95: ${p95}ms | P99: ${p99}ms | Success: ${success}%"
    fi
}

# =============================================================================
# Generate Report
# =============================================================================
generate_report() {
    echo -e "\n${YELLOW}Generating report...${NC}"
    
    cat > "$REPORT_FILE" << EOF
# SimpleEmailAPI Performance Benchmark Report

**Generated:** $(date '+%Y-%m-%d %H:%M:%S %Z')  
**Host:** $PERF_API_HOST  
**Mode:** $([ -n "$DRY_RUN" ] && echo "Dry-Run (simulated SES delay)" || echo "Live")

---

## Summary

| Protocol | Mode | RPS | P50 (ms) | P95 (ms) | P99 (ms) | Success |
|----------|------|-----|----------|----------|----------|---------|
EOF

    # Add HTTP results
    for rps in 10 50 100; do
        local key_prefix="http_sync_${rps}"
        if [[ -n "${RESULTS[${key_prefix}_p50]}" ]]; then
            echo "| HTTP | Sync | $rps | ${RESULTS[${key_prefix}_p50]} | ${RESULTS[${key_prefix}_p95]} | ${RESULTS[${key_prefix}_p99]} | ${RESULTS[${key_prefix}_success]}% |" >> "$REPORT_FILE"
        fi
    done
    
    for rps in 10 50 100; do
        local key_prefix="http_async_${rps}"
        if [[ -n "${RESULTS[${key_prefix}_p50]}" ]]; then
            echo "| HTTP | Async | $rps | ${RESULTS[${key_prefix}_p50]} | ${RESULTS[${key_prefix}_p95]} | ${RESULTS[${key_prefix}_p99]} | ${RESULTS[${key_prefix}_success]}% |" >> "$REPORT_FILE"
        fi
    done
    
    # Add gRPC results
    for rps in 10 50 100; do
        local key_prefix="grpc_sync_${rps}"
        if [[ -n "${RESULTS[${key_prefix}_p50]}" ]]; then
            echo "| gRPC | Sync | $rps | ${RESULTS[${key_prefix}_p50]} | ${RESULTS[${key_prefix}_p95]} | ${RESULTS[${key_prefix}_p99]} | ${RESULTS[${key_prefix}_success]}% |" >> "$REPORT_FILE"
        fi
    done
    
    for rps in 10 50 100; do
        local key_prefix="grpc_async_${rps}"
        if [[ -n "${RESULTS[${key_prefix}_p50]}" ]]; then
            echo "| gRPC | Async | $rps | ${RESULTS[${key_prefix}_p50]} | ${RESULTS[${key_prefix}_p95]} | ${RESULTS[${key_prefix}_p99]} | ${RESULTS[${key_prefix}_success]}% |" >> "$REPORT_FILE"
        fi
    done

    cat >> "$REPORT_FILE" << 'EOF'

---

## Analysis

### Key Findings

- **Sync Mode**: Includes SES latency (~50-150ms). Use for measuring end-to-end email send time.
- **Async Mode**: Returns immediately after queueing. Use for measuring pure API throughput.
- **P99/P50 Ratio**: If > 5x, indicates tail latency issues (connection pool, GC, DB contention).

### Recommended Thresholds

| Metric | Good | Warning | Critical |
|--------|------|---------|----------|
| P50 | < 100ms | < 200ms | > 300ms |
| P95 | < 300ms | < 500ms | > 800ms |
| P99 | < 500ms | < 800ms | > 1500ms |
| Success | > 99.5% | > 99% | < 99% |

---

## Test Configuration

- **Sync Mode**: `async: false` - Waits for SES response
- **Async Mode**: `async: true` - Returns after queue insertion
- **RPS Levels**: 10 (baseline), 50 (moderate), 100 (high load)
EOF

    echo -e "${GREEN}✓ Report saved to: $REPORT_FILE${NC}"
}

# =============================================================================
# Main
# =============================================================================
main() {
    load_config
    check_tools
    mkdir -p "$RESULTS_DIR"
    
    print_banner
    
    # Determine test duration
    local duration="15s"
    [[ -n "$QUICK_MODE" ]] && duration="5s"
    
    echo -e "${BOLD}Starting benchmark suite...${NC}\n"
    
    # HTTP Tests
    if [[ -z "$GRPC_ONLY" ]]; then
        echo -e "${YELLOW}━━━ HTTP Endpoint Tests ━━━${NC}"
        
        # Sync mode at different RPS
        run_http_sync_test "baseline" 10 "$duration"
        run_http_sync_test "moderate" 50 "$duration"
        run_http_sync_test "high" 100 "$duration"
        
        echo ""
        
        # Async mode at different RPS
        run_http_async_test "baseline" 10 "$duration"
        run_http_async_test "moderate" 50 "$duration"
        run_http_async_test "high" 100 "$duration"
        
        echo ""
    fi
    
    # gRPC Tests
    if [[ -z "$HTTP_ONLY" ]]; then
        echo -e "${YELLOW}━━━ gRPC Endpoint Tests ━━━${NC}"
        
        # Sync mode
        run_grpc_test "sync" 10 "$duration"
        run_grpc_test "sync" 50 "$duration"
        run_grpc_test "sync" 100 "$duration"
        
        echo ""
        
        # Async mode
        run_grpc_test "async" 10 "$duration"
        run_grpc_test "async" 50 "$duration"
        run_grpc_test "async" 100 "$duration"
        
        echo ""
    fi
    
    # Generate report
    generate_report
    
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}${BOLD}                    Benchmark Complete!                    ${NC}${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "View full report: ${BLUE}$REPORT_FILE${NC}"
    echo ""
    
    # Print quick summary table
    echo -e "${BOLD}Quick Summary:${NC}"
    echo ""
    printf "%-8s %-6s %8s %8s %8s %8s\n" "Protocol" "Mode" "P50" "P95" "P99" "Success"
    printf "%-8s %-6s %8s %8s %8s %8s\n" "--------" "------" "--------" "--------" "--------" "--------"
    
    for proto in http grpc; do
        for mode in sync async; do
            local key="${proto}_${mode}_50"
            if [[ -n "${RESULTS[${key}_p50]}" ]]; then
                printf "%-8s %-6s %6sms %6sms %6sms %7s%%\n" \
                    "${proto^^}" "${mode^}" \
                    "${RESULTS[${key}_p50]}" \
                    "${RESULTS[${key}_p95]}" \
                    "${RESULTS[${key}_p99]}" \
                    "${RESULTS[${key}_success]}"
            fi
        done
    done
    echo ""
}

main "$@"
