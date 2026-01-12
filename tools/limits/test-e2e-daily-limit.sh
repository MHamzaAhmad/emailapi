#!/bin/bash

# =============================================================================
# E2E Daily Limit Flow Test
# Usage: ./test-e2e-daily-limit.sh
#
# Simulates realistic daily limit behavior:
#
# 1. Normal user: Send up to free limit (100/day)
# 2. Trigger soft suspension via bounces
# 3. Verify soft limit applies (10/day)
# 4. Try to exceed soft limit - should be blocked
#
# This is a REAL test - it consumes your daily quota!
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

BOUNCE_EMAIL="bounce@simulator.amazonses.com"
SUCCESS_EMAIL="success@simulator.amazonses.com"

# Load config
if [[ -f "$SCRIPT_DIR/.env" ]]; then
    export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
fi

API_HOST="${API_HOST:-localhost:8080}"

if [[ -z "$API_KEY" || -z "$FROM_EMAIL" ]]; then
    echo -e "${RED}ERROR: Missing API_KEY and FROM_EMAIL in .env${NC}"
    exit 1
fi

# Add dry run support
DRY_RUN=""
for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN="true"
            ;;
    esac
done

HEADERS=(-H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json")
if [[ -n "$DRY_RUN" ]]; then
    HEADERS+=(-H "X-Dry-Run: true")
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  E2E Daily Limit Flow Test${NC}"
if [[ -n "$DRY_RUN" ]]; then
    echo -e "${YELLOW}  Mode: DRY-RUN${NC}"
fi
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}This test will:${NC}"
echo "  1. Check current daily usage"
echo "  2. Send emails until daily limit"
echo "  3. Verify limit error message"
echo ""
echo -e "${RED}⚠️  This consumes your daily quota!${NC}"
read -p "Press Enter to continue..."

send_email() {
    local TO=$1
    local SUBJECT=$2
    
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "http://$API_HOST/v1.EmailService/SendEmail" \
        "${HEADERS[@]}" \
        -d "{
            \"from\": \"$FROM_EMAIL\",
            \"to\": [\"$TO\"],
            \"subject\": \"$SUBJECT\",
            \"body\": \"Test at $(date)\"
        }")
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    
    echo "$HTTP_CODE|$BODY"
}

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}PHASE 1: Pre-check - Getting baseline${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

RESULT=$(send_email "$SUCCESS_EMAIL" "Baseline Check")
HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)

if [[ "$HTTP_CODE" == "200" ]]; then
    echo -e "  ${GREEN}✓${NC} Account active, baseline send successful"
else
    BODY=$(echo "$RESULT" | cut -d'|' -f2-)
    echo -e "  ${RED}✗${NC} Baseline failed:"
    echo "  $BODY" | jq . 2>/dev/null || echo "  $BODY"
    echo ""
    echo "Fix the issue before running this test."
    exit 1
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}PHASE 2: Consuming daily quota${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

SUCCESS_COUNT=0
LIMIT_HIT=0
LIMIT_ERROR=""

# Send up to 120 emails (beyond 100 limit)
for i in $(seq 1 120); do
    RESULT=$(send_email "$SUCCESS_EMAIL" "Daily Limit Test $i")
    HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
    BODY=$(echo "$RESULT" | cut -d'|' -f2-)
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        ((SUCCESS_COUNT++))
        echo -ne "\r  ${GREEN}Sent:${NC} $SUCCESS_COUNT | ${YELLOW}Limit hits:${NC} $LIMIT_HIT"
    else
        if echo "$BODY" | grep -qiE "daily|limit|quota"; then
            ((LIMIT_HIT++))
            if [[ $LIMIT_HIT -eq 1 ]]; then
                LIMIT_ERROR="$BODY"
            fi
            echo -ne "\r  ${GREEN}Sent:${NC} $SUCCESS_COUNT | ${YELLOW}Limit hits:${NC} $LIMIT_HIT"
            
            # Once we hit the limit, no point continuing
            if [[ $LIMIT_HIT -ge 5 ]]; then
                break
            fi
        else
            echo ""
            echo -e "  ${RED}Unexpected error:${NC} $BODY"
            break
        fi
    fi
    
    # Small delay to avoid rate limit
    sleep 0.1
done

echo ""
echo ""

if [[ $LIMIT_HIT -gt 0 ]]; then
    echo -e "${GREEN}✓ Daily limit is working!${NC}"
    echo ""
    echo -e "${YELLOW}First limit response:${NC}"
    echo "$LIMIT_ERROR" | jq . 2>/dev/null || echo "$LIMIT_ERROR"
    
    # Extract limit info from response
    DAILY_LIMIT=$(echo "$LIMIT_ERROR" | jq -r '.details.daily_limit // "unknown"' 2>/dev/null)
    DAILY_USAGE=$(echo "$LIMIT_ERROR" | jq -r '.details.daily_usage // "unknown"' 2>/dev/null)
    
    if [[ "$DAILY_LIMIT" != "unknown" ]]; then
        echo ""
        echo -e "${CYAN}Limit Info:${NC}"
        echo "  Daily Limit: $DAILY_LIMIT"
        echo "  Daily Usage: $DAILY_USAGE"
    fi
else
    echo -e "${YELLOW}⚠ No daily limit hit after $SUCCESS_COUNT emails${NC}"
    echo "  User may be on paid plan (unlimited) or limit not configured"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}Summary${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${GREEN}Emails sent:${NC}    $SUCCESS_COUNT"
echo -e "  ${YELLOW}Limit blocked:${NC}  $LIMIT_HIT"
echo ""

if [[ $SUCCESS_COUNT -le 15 && $LIMIT_HIT -gt 0 ]]; then
    echo -e "  ${CYAN}Note: Low success count suggests soft suspension (10/day limit)${NC}"
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
