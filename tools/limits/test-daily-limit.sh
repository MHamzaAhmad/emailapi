#!/bin/bash

# =============================================================================
# Daily Limit Testing
# Usage: ./test-daily-limit.sh [--dry-run] [--count N]
#
# Sends requests to exhaust daily limit for free users (default: 100/day).
# Expects limit errors after hitting the quota.
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# Defaults
DRY_RUN=""
REQUEST_COUNT=110  # Over the 100/day limit

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
    exit 1
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Daily Limit Test${NC}"
if [[ -n "$DRY_RUN" ]]; then
    echo -e "${YELLOW}  Mode: DRY-RUN (no emails sent)${NC}"
fi
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Host: $API_HOST"
echo "  Requests: $REQUEST_COUNT"
echo ""
echo -e "${YELLOW}Note: This will consume your daily quota!${NC}"
echo ""

# Build headers
HEADERS=(-H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json")
if [[ -n "$DRY_RUN" ]]; then
    HEADERS+=(-H "X-Dry-Run: true")
fi

# Counters
SUCCESS=0
DAILY_LIMITED=0
OTHER_ERRORS=0

echo -e "${YELLOW}Sending $REQUEST_COUNT requests (with 100ms delay to avoid rate limit)...${NC}"
echo ""

for i in $(seq 1 $REQUEST_COUNT); do
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "http://$API_HOST/v1.EmailService/SendEmail" \
        "${HEADERS[@]}" \
        -d "{
            \"from\": \"$FROM_EMAIL\",
            \"to\": [\"success@simulator.amazonses.com\"],
            \"subject\": \"Daily Limit Test $i\",
            \"body\": \"Testing daily limit\"
        }")
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    
    case $HTTP_CODE in
        200)
            ((SUCCESS++))
            echo -ne "\r  ${GREEN}✓${NC} Sent: $SUCCESS | ${YELLOW}Daily Limited: $DAILY_LIMITED${NC}"
            ;;
        *)
            # Check if it's a daily limit error
            if echo "$BODY" | grep -qi "daily\|limit"; then
                ((DAILY_LIMITED++))
                echo -ne "\r  ${GREEN}✓${NC} Sent: $SUCCESS | ${YELLOW}Daily Limited: $DAILY_LIMITED${NC}"
                
                if [[ $DAILY_LIMITED -eq 1 ]]; then
                    echo ""
                    echo -e "  ${YELLOW}Daily limit response:${NC}"
                    echo "  $BODY" | jq -r '.message // .' 2>/dev/null || echo "  $BODY"
                fi
            else
                ((OTHER_ERRORS++))
            fi
            ;;
    esac
    
    # Small delay to avoid hitting rate limit
    sleep 0.1
done

echo ""
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Results${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${GREEN}Successful:${NC}     $SUCCESS"
echo -e "  ${YELLOW}Daily Limited:${NC}  $DAILY_LIMITED"
echo -e "  ${RED}Other Errors:${NC}   $OTHER_ERRORS"
echo ""

if [[ $DAILY_LIMITED -gt 0 ]]; then
    echo -e "${GREEN}✓ Daily limit is working!${NC}"
else
    echo -e "${YELLOW}⚠ No daily limit hit - user may be on paid plan or limit not configured${NC}"
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
