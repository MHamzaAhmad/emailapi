#!/bin/bash

# =============================================================================
# Rate Limit Testing
# Usage: ./test-rate-limit.sh [--dry-run] [--count N]
#
# Sends rapid requests to test rate limiting behavior.
# Expects 429 responses after hitting the limit.
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

# Defaults
DRY_RUN=""
REQUEST_COUNT=120  # Default: slightly over typical 100/min limit

# Parse args
for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN="true"
            ;;
        --count=*)
            REQUEST_COUNT="${arg#*=}"
            ;;
    esac
done

# Load config
if [[ -f "$SCRIPT_DIR/.env" ]]; then
    export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
fi

API_HOST="${API_HOST:-localhost:8080}"

if [[ -z "$API_KEY" || -z "$FROM_EMAIL" ]]; then
    echo -e "${RED}ERROR: Missing required environment variables${NC}"
    echo "Required: API_KEY, FROM_EMAIL"
    echo "Copy .env.example to .env and configure"
    exit 1
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Rate Limit Test${NC}"
if [[ -n "$DRY_RUN" ]]; then
    echo -e "${YELLOW}  Mode: DRY-RUN (no emails sent)${NC}"
fi
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Host: $API_HOST"
echo "  Requests: $REQUEST_COUNT"
echo ""

# Build headers
HEADERS=(-H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json")
if [[ -n "$DRY_RUN" ]]; then
    HEADERS+=(-H "X-Dry-Run: true")
fi

# Counters
SUCCESS=0
RATE_LIMITED=0
OTHER_ERRORS=0

echo -e "${YELLOW}Sending $REQUEST_COUNT requests...${NC}"
echo ""

for i in $(seq 1 $REQUEST_COUNT); do
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "http://$API_HOST/v1.EmailService/SendEmail" \
        "${HEADERS[@]}" \
        -d "{
            \"from\": \"$FROM_EMAIL\",
            \"to\": [\"success@simulator.amazonses.com\"],
            \"subject\": \"Rate Limit Test $i\",
            \"body\": \"Testing rate limit\"
        }")
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    
    case $HTTP_CODE in
        200)
            ((SUCCESS++))
            echo -ne "\r  ${GREEN}✓${NC} Sent: $SUCCESS | ${YELLOW}Rate Limited: $RATE_LIMITED${NC} | ${RED}Errors: $OTHER_ERRORS${NC}"
            ;;
        429)
            ((RATE_LIMITED++))
            echo -ne "\r  ${GREEN}✓${NC} Sent: $SUCCESS | ${YELLOW}Rate Limited: $RATE_LIMITED${NC} | ${RED}Errors: $OTHER_ERRORS${NC}"
            
            # Show first rate limit response
            if [[ $RATE_LIMITED -eq 1 ]]; then
                echo ""
                echo -e "  ${YELLOW}First 429 response:${NC}"
                BODY=$(echo "$RESPONSE" | sed '$d')
                echo "  $BODY" | jq -r '.message // .' 2>/dev/null || echo "  $BODY"
            fi
            ;;
        *)
            ((OTHER_ERRORS++))
            echo -ne "\r  ${GREEN}✓${NC} Sent: $SUCCESS | ${YELLOW}Rate Limited: $RATE_LIMITED${NC} | ${RED}Errors: $OTHER_ERRORS${NC}"
            ;;
    esac
done

echo ""
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Results${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${GREEN}Successful:${NC}    $SUCCESS"
echo -e "  ${YELLOW}Rate Limited:${NC}  $RATE_LIMITED"
echo -e "  ${RED}Other Errors:${NC}  $OTHER_ERRORS"
echo ""

if [[ $RATE_LIMITED -gt 0 ]]; then
    echo -e "${GREEN}✓ Rate limiting is working!${NC}"
else
    echo -e "${RED}✗ No rate limiting detected - check configuration${NC}"
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
