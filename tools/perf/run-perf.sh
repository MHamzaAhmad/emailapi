#!/bin/bash

# =============================================================================
# Performance Testing Runner
# Usage: ./run-perf.sh [smoke|load|stress] [grpc|http|both] [--dry-run]
#
# Options:
#   --dry-run    Test API without sending actual emails (saves SES quota)
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"
PROTO_DIR="$SCRIPT_DIR/../../proto"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Dry-run flag
DRY_RUN=""

# =============================================================================
# Configuration
# =============================================================================
load_config() {
    # Load from .env if exists
    if [[ -f "$SCRIPT_DIR/.env" ]]; then
        export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
    fi

    # Validate required vars
    if [[ -z "$PERF_API_KEY" ]]; then
        echo -e "${RED}ERROR: PERF_API_KEY is required${NC}"
        echo "Set it via: export PERF_API_KEY=your-key"
        echo "Or create tools/perf/.env with PERF_API_KEY=your-key"
        exit 1
    fi

    if [[ -z "$PERF_FROM_EMAIL" ]]; then
        echo -e "${RED}ERROR: PERF_FROM_EMAIL is required${NC}"
        exit 1
    fi

    if [[ -z "$PERF_TO_EMAIL" ]]; then
        echo -e "${RED}ERROR: PERF_TO_EMAIL is required${NC}"
        exit 1
    fi

    # Defaults
    PERF_API_HOST="${PERF_API_HOST:-api.simpleemailapi.dev}"
    PERF_GRPC_PORT="${PERF_GRPC_PORT:-443}"
    PERF_HTTP_PORT="${PERF_HTTP_PORT:-443}"
}

# =============================================================================
# Utility Functions
# =============================================================================
check_tool() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}ERROR: $1 is not installed${NC}"
        echo "Install with: $2"
        exit 1
    fi
}

print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    if [[ -n "$DRY_RUN" ]]; then
        echo -e "${YELLOW}  Mode: DRY-RUN (no emails sent)${NC}"
    fi
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
}

print_config() {
    echo -e "${YELLOW}Configuration:${NC}"
    echo "  Host:       $PERF_API_HOST"
    echo "  gRPC Port:  $PERF_GRPC_PORT"
    echo "  HTTP Port:  $PERF_HTTP_PORT"
    echo "  From:       $PERF_FROM_EMAIL"
    echo "  To:         $PERF_TO_EMAIL"
    if [[ -n "$DRY_RUN" ]]; then
        echo "  Dry-Run:    YES (no actual emails sent)"
    fi
    echo ""
}

# Build metadata JSON with optional dry-run header
get_grpc_metadata() {
    if [[ -n "$DRY_RUN" ]]; then
        echo "{\"authorization\": \"Bearer $PERF_API_KEY\", \"x-dry-run\": \"true\"}"
    else
        echo "{\"authorization\": \"Bearer $PERF_API_KEY\"}"
    fi
}

# =============================================================================
# gRPC Tests (using ghz)
# =============================================================================
run_grpc_smoke() {
    print_header "gRPC Smoke Test (10 RPS, 30s)"
    
    local output_file="$RESULTS_DIR/grpc-smoke-$(date +%Y%m%d-%H%M%S).json"
    local metadata=$(get_grpc_metadata)
    
    ghz --insecure="${GRPC_INSECURE:-false}" \
        --proto="$PROTO_DIR/public/v1/email.proto" \
        --import-paths="$PROTO_DIR/public" \
        --call=v1.EmailService/SendEmail \
        --rps=10 \
        --duration=30s \
        --connections=5 \
        --concurrency=10 \
        --metadata="$metadata" \
        --data="{\"from\": \"$PERF_FROM_EMAIL\", \"to\": [\"$PERF_TO_EMAIL\"], \"subject\": \"gRPC Smoke Test $(date +%s)\", \"body\": \"Performance smoke test via gRPC\"}" \
        --format=pretty \
        --output="$output_file" \
        "$PERF_API_HOST:$PERF_GRPC_PORT"
    
    echo -e "\n${GREEN}Results saved to: $output_file${NC}"
}

run_grpc_load() {
    print_header "gRPC Load Test (100 RPS, 2m)"
    
    local output_file="$RESULTS_DIR/grpc-load-$(date +%Y%m%d-%H%M%S).json"
    local metadata=$(get_grpc_metadata)
    
    ghz --insecure="${GRPC_INSECURE:-false}" \
        --proto="$PROTO_DIR/public/v1/email.proto" \
        --import-paths="$PROTO_DIR/public" \
        --call=v1.EmailService/SendEmail \
        --rps=100 \
        --duration=2m \
        --connections=20 \
        --concurrency=50 \
        --metadata="$metadata" \
        --data="{\"from\": \"$PERF_FROM_EMAIL\", \"to\": [\"$PERF_TO_EMAIL\"], \"subject\": \"gRPC Load Test $(date +%s)\", \"body\": \"Performance load test via gRPC\"}" \
        --format=pretty \
        --output="$output_file" \
        "$PERF_API_HOST:$PERF_GRPC_PORT"
    
    echo -e "\n${GREEN}Results saved to: $output_file${NC}"
}

run_grpc_stress() {
    print_header "gRPC Stress Test (50 → 500 RPS, 5m)"
    
    local output_file="$RESULTS_DIR/grpc-stress-$(date +%Y%m%d-%H%M%S).json"
    local metadata=$(get_grpc_metadata)
    
    # Stress test: start at 50 RPS, end at 500 RPS over 5 minutes
    # ghz doesn't support ramping natively, so we run multiple stages
    
    echo -e "${YELLOW}Stage 1/5: 50 RPS (1m)${NC}"
    ghz --insecure="${GRPC_INSECURE:-false}" \
        --proto="$PROTO_DIR/public/v1/email.proto" \
        --import-paths="$PROTO_DIR/public" \
        --call=v1.EmailService/SendEmail \
        --rps=50 \
        --duration=1m \
        --connections=30 \
        --concurrency=100 \
        --metadata="$metadata" \
        --data="{\"from\": \"$PERF_FROM_EMAIL\", \"to\": [\"$PERF_TO_EMAIL\"], \"subject\": \"gRPC Stress S1 $(date +%s)\", \"body\": \"Stress test stage 1\"}" \
        --format=summary \
        "$PERF_API_HOST:$PERF_GRPC_PORT"
    
    echo -e "${YELLOW}Stage 2/5: 150 RPS (1m)${NC}"
    ghz --insecure="${GRPC_INSECURE:-false}" \
        --proto="$PROTO_DIR/public/v1/email.proto" \
        --import-paths="$PROTO_DIR/public" \
        --call=v1.EmailService/SendEmail \
        --rps=150 \
        --duration=1m \
        --connections=50 \
        --concurrency=150 \
        --metadata="$metadata" \
        --data="{\"from\": \"$PERF_FROM_EMAIL\", \"to\": [\"$PERF_TO_EMAIL\"], \"subject\": \"gRPC Stress S2 $(date +%s)\", \"body\": \"Stress test stage 2\"}" \
        --format=summary \
        "$PERF_API_HOST:$PERF_GRPC_PORT"
    
    echo -e "${YELLOW}Stage 3/5: 250 RPS (1m)${NC}"
    ghz --insecure="${GRPC_INSECURE:-false}" \
        --proto="$PROTO_DIR/public/v1/email.proto" \
        --import-paths="$PROTO_DIR/public" \
        --call=v1.EmailService/SendEmail \
        --rps=250 \
        --duration=1m \
        --connections=70 \
        --concurrency=200 \
        --metadata="$metadata" \
        --data="{\"from\": \"$PERF_FROM_EMAIL\", \"to\": [\"$PERF_TO_EMAIL\"], \"subject\": \"gRPC Stress S3 $(date +%s)\", \"body\": \"Stress test stage 3\"}" \
        --format=summary \
        "$PERF_API_HOST:$PERF_GRPC_PORT"
    
    echo -e "${YELLOW}Stage 4/5: 350 RPS (1m)${NC}"
    ghz --insecure="${GRPC_INSECURE:-false}" \
        --proto="$PROTO_DIR/public/v1/email.proto" \
        --import-paths="$PROTO_DIR/public" \
        --call=v1.EmailService/SendEmail \
        --rps=350 \
        --duration=1m \
        --connections=80 \
        --concurrency=250 \
        --metadata="$metadata" \
        --data="{\"from\": \"$PERF_FROM_EMAIL\", \"to\": [\"$PERF_TO_EMAIL\"], \"subject\": \"gRPC Stress S4 $(date +%s)\", \"body\": \"Stress test stage 4\"}" \
        --format=summary \
        "$PERF_API_HOST:$PERF_GRPC_PORT"
    
    echo -e "${YELLOW}Stage 5/5: 500 RPS (1m)${NC}"
    ghz --insecure="${GRPC_INSECURE:-false}" \
        --proto="$PROTO_DIR/public/v1/email.proto" \
        --import-paths="$PROTO_DIR/public" \
        --call=v1.EmailService/SendEmail \
        --rps=500 \
        --duration=1m \
        --connections=100 \
        --concurrency=300 \
        --metadata="$metadata" \
        --data="{\"from\": \"$PERF_FROM_EMAIL\", \"to\": [\"$PERF_TO_EMAIL\"], \"subject\": \"gRPC Stress S5 $(date +%s)\", \"body\": \"Stress test stage 5\"}" \
        --format=pretty \
        --output="$output_file" \
        "$PERF_API_HOST:$PERF_GRPC_PORT"
    
    echo -e "\n${GREEN}Final stage results saved to: $output_file${NC}"
}

# =============================================================================
# HTTP Tests (using k6)
# =============================================================================
run_http_test() {
    local test_type=$1
    local script="$SCRIPT_DIR/http/${test_type}.js"
    local output_file="$RESULTS_DIR/http-${test_type}-$(date +%Y%m%d-%H%M%S).json"
    
    print_header "HTTP ${test_type^} Test"
    
    local dry_run_env=""
    if [[ -n "$DRY_RUN" ]]; then
        dry_run_env="--env DRY_RUN=true"
    fi
    
    k6 run \
        --env API_KEY="$PERF_API_KEY" \
        --env API_HOST="$PERF_API_HOST" \
        --env API_PORT="$PERF_HTTP_PORT" \
        --env FROM_EMAIL="$PERF_FROM_EMAIL" \
        --env TO_EMAIL="$PERF_TO_EMAIL" \
        $dry_run_env \
        --out json="$output_file" \
        "$script"
    
    echo -e "\n${GREEN}Results saved to: $output_file${NC}"
}

# =============================================================================
# Main
# =============================================================================
main() {
    local test_type="smoke"
    local protocol="both"
    
    # Parse arguments
    for arg in "$@"; do
        case $arg in
            smoke|load|stress)
                test_type="$arg"
                ;;
            grpc|http|both)
                protocol="$arg"
                ;;
            --dry-run)
                DRY_RUN="true"
                ;;
            --help|-h)
                echo "Usage: $0 [smoke|load|stress] [grpc|http|both] [--dry-run]"
                echo ""
                echo "Test Types:"
                echo "  smoke   - Baseline test (10 RPS, 30s)"
                echo "  load    - Realistic traffic (100 RPS, 2m)"
                echo "  stress  - Find breaking point (50-500 RPS, 5m)"
                echo ""
                echo "Protocols:"
                echo "  grpc    - Test gRPC endpoint only"
                echo "  http    - Test HTTP endpoint only"
                echo "  both    - Test both (default)"
                echo ""
                echo "Options:"
                echo "  --dry-run  Test without sending actual emails (saves SES quota)"
                exit 0
                ;;
        esac
    done
    
    load_config
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    print_header "SimpleEmailAPI Performance Test Suite"
    print_config
    
    # Check required tools
    if [[ "$protocol" == "grpc" || "$protocol" == "both" ]]; then
        check_tool "ghz" "brew install ghz (macOS) or go install github.com/bojand/ghz/cmd/ghz@latest"
    fi
    
    if [[ "$protocol" == "http" || "$protocol" == "both" ]]; then
        check_tool "k6" "brew install k6 (macOS) or see https://k6.io/docs/get-started/installation/"
    fi
    
    # Run tests
    case "$test_type" in
        smoke)
            [[ "$protocol" == "grpc" || "$protocol" == "both" ]] && run_grpc_smoke
            [[ "$protocol" == "http" || "$protocol" == "both" ]] && run_http_test smoke
            ;;
        load)
            [[ "$protocol" == "grpc" || "$protocol" == "both" ]] && run_grpc_load
            [[ "$protocol" == "http" || "$protocol" == "both" ]] && run_http_test load
            ;;
        stress)
            [[ "$protocol" == "grpc" || "$protocol" == "both" ]] && run_grpc_stress
            [[ "$protocol" == "http" || "$protocol" == "both" ]] && run_http_test stress
            ;;
    esac
    
    echo ""
    echo -e "${GREEN}✓ Performance tests completed!${NC}"
    echo -e "Results are in: ${BLUE}$RESULTS_DIR${NC}"
}

main "$@"
