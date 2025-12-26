#!/bin/bash

# =============================================================================
# Quick Single Request Test
# Usage: ./quick-test.sh [grpc|http] [--dry-run]
# 
# Sends a single request to measure baseline latency without load testing.
# Great for verifying your API key and connectivity before running full tests.
#
# With --dry-run: Tests full API path without actually sending emails (saves SES quota)
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_DIR="$SCRIPT_DIR/../../proto"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# Parse args
PROTOCOL="both"
DRY_RUN=""

for arg in "$@"; do
    case $arg in
        grpc|http|both)
            PROTOCOL="$arg"
            ;;
        --dry-run)
            DRY_RUN="true"
            ;;
    esac
done

# Load config
if [[ -f "$SCRIPT_DIR/.env" ]]; then
    export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
fi

PERF_API_HOST="${PERF_API_HOST:-api.simpleemailapi.dev}"
PERF_GRPC_PORT="${PERF_GRPC_PORT:-443}"
PERF_HTTP_PORT="${PERF_HTTP_PORT:-443}"

if [[ -z "$PERF_API_KEY" || -z "$PERF_FROM_EMAIL" || -z "$PERF_TO_EMAIL" ]]; then
    echo -e "${RED}ERROR: Missing required environment variables${NC}"
    echo "Required: PERF_API_KEY, PERF_FROM_EMAIL, PERF_TO_EMAIL"
    echo "Set them in tools/perf/.env or export them"
    exit 1
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Quick Latency Test${NC}"
if [[ -n "$DRY_RUN" ]]; then
    echo -e "${YELLOW}  Mode: DRY-RUN (no emails sent)${NC}"
fi
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Host: $PERF_API_HOST"
echo "  From: $PERF_FROM_EMAIL"
echo "  To:   $PERF_TO_EMAIL"
echo ""

# HTTP Test
if [[ "$PROTOCOL" == "http" || "$PROTOCOL" == "both" ]]; then
    echo -e "${YELLOW}Testing HTTP endpoint...${NC}"
    
    # Build headers
    HEADERS=(-H "Authorization: Bearer $PERF_API_KEY" -H "Content-Type: application/json")
    if [[ -n "$DRY_RUN" ]]; then
        HEADERS+=(-H "X-Dry-Run: true")
    fi
    
    START_TIME=$(python3 -c 'import time; print(int(time.time() * 1000))')
    
    RESPONSE=$(curl -s -w "\n%{http_code}\n%{time_total}" \
        -X POST "https://$PERF_API_HOST/v1.EmailService/SendEmail" \
        "${HEADERS[@]}" \
        -d "{
            \"from\": \"$PERF_FROM_EMAIL\",
            \"to\": [\"$PERF_TO_EMAIL\"],
            \"subject\": \"Quick Test $(date +%s)\",
            \"body\": \"Quick latency test at $(date)\"
        }")
    
    # Parse response
    HTTP_CODE=$(echo "$RESPONSE" | tail -2 | head -1)
    TOTAL_TIME=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | head -n -2)
    
    LATENCY_MS=$(echo "$TOTAL_TIME * 1000" | bc | cut -d'.' -f1)
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo -e "  ${GREEN}✓${NC} HTTP Status: $HTTP_CODE"
        echo -e "  ${GREEN}✓${NC} Latency: ${LATENCY_MS}ms"
        echo "  Response: $BODY"
    else
        echo -e "  ${RED}✗${NC} HTTP Status: $HTTP_CODE"
        echo "  Response: $BODY"
    fi
    echo ""
fi

# gRPC Test
if [[ "$PROTOCOL" == "grpc" || "$PROTOCOL" == "both" ]]; then
    echo -e "${YELLOW}Testing gRPC endpoint...${NC}"
    
    if ! command -v grpcurl &> /dev/null; then
        echo -e "  ${RED}✗${NC} grpcurl not installed (brew install grpcurl)"
    else
        # Build headers
        GRPC_HEADERS="-H \"authorization: Bearer $PERF_API_KEY\""
        if [[ -n "$DRY_RUN" ]]; then
            GRPC_HEADERS="$GRPC_HEADERS -H \"x-dry-run: true\""
        fi
        
        START_TIME=$(python3 -c 'import time; print(int(time.time() * 1000))')
        
        RESPONSE=$(eval grpcurl \
            -proto "$PROTO_DIR/public/v1/email.proto" \
            -import-path "$PROTO_DIR/public" \
            $GRPC_HEADERS \
            -d "{
                \"from\": \"$PERF_FROM_EMAIL\",
                \"to\": [\"$PERF_TO_EMAIL\"],
                \"subject\": \"gRPC Quick Test $(date +%s)\",
                \"body\": \"gRPC latency test at $(date)\"
            }" \
            "$PERF_API_HOST:$PERF_GRPC_PORT" \
            v1.EmailService/SendEmail 2>&1)
        
        END_TIME=$(python3 -c 'import time; print(int(time.time() * 1000))')
        LATENCY_MS=$((END_TIME - START_TIME))
        
        if echo "$RESPONSE" | grep -q '"id":'; then
            echo -e "  ${GREEN}✓${NC} gRPC Status: OK"
            echo -e "  ${GREEN}✓${NC} Latency: ${LATENCY_MS}ms"
            echo "  Response: $RESPONSE"
        else
            echo -e "  ${RED}✗${NC} gRPC Error:"
            echo "  $RESPONSE"
        fi
    fi
    echo ""
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
