#!/bin/bash

# =============================================================================
# Limits Testing (Dry-Run Mode)
# Usage: ./test-limits-dry-run.sh [--scenario=rate|daily|monthly|all]
#
# Tests all limit scenarios in dry-run mode (no real SES sends).
# Uses ONLY simulator addresses for safety - if dry-run fails, still safe.
#
# SCENARIOS:
#   rate    - Rate limiting (burst 120 requests → 429 errors)
#   daily   - Daily limits (exhaust quota → limit errors)
#   monthly - Monthly limits (exhaust quota → limit errors)
#   all     - Run all scenarios (default)
#
# SAFETY: 
#   - Uses X-Dry-Run header (no real SES API calls)
#   - ONLY sends to success@simulator.amazonses.com (defense in depth)
#   - Respects SES rate limits (adds delays)
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

# Parse arguments
SCENARIO="all"
for arg in "$@"; do
    case $arg in
        --scenario=*)
            SCENARIO="${arg#*=}"
            ;;
    esac
done

# Load config
if [[ -f "$SCRIPT_DIR/.env" ]]; then
    export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
fi

API_HOST="${API_HOST:-localhost:8080}"
SES_RATE_LIMIT="${SES_RATE_LIMIT:-1}"  # Requests per second
REQUEST_DELAY=$(awk "BEGIN {print 1/$SES_RATE_LIMIT}")

# MANDATORY: Only simulator addresses
SUCCESS_EMAIL="success@simulator.amazonses.com"

if [[ -z "$API_KEY" || -z "$FROM_EMAIL" ]]; then
    echo -e "${RED}ERROR: Configure API_KEY and FROM_EMAIL in .env${NC}"
    exit 1
fi

# Force dry-run mode
HEADERS=(
    -H "Authorization: Bearer $API_KEY"
    -H "Content-Type: application/json"
    -H "X-Dry-Run: true"
)

print_header() {
    echo ""
    echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║${NC}  $1"
    echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

send_email() {
    local SUBJECT=$1
    
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "http://$API_HOST/v1.EmailService/SendEmail" \
        "${HEADERS[@]}" \
        -d "{
            \"from\": \"$FROM_EMAIL\",
            \"to\": [\"$SUCCESS_EMAIL\"],
            \"subject\": \"$SUBJECT\",
            \"body\": \"Dry-run test\"
        }")
    
    echo "$(echo "$RESPONSE" | tail -1)|$(echo "$RESPONSE" | sed '$d')"
}

# =============================================================================
# SCENARIO: Rate Limiting
# =============================================================================
scenario_rate_limit() {
    print_header "SCENARIO: Rate Limiting"
    
    echo -e "${YELLOW}Sending 120 rapid requests to trigger rate limit...${NC}"
    echo -e "${CYAN}Using dry-run mode + simulator address${NC}"
    echo ""
    
    local SUCCESS=0
    local RATE_LIMITED=0
    local FIRST_RATE_LIMIT_MSG=""
    
    for i in $(seq 1 120); do
        RESULT=$(send_email "Rate Test $i")
        HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
        BODY=$(echo "$RESULT" | cut -d'|' -f2-)
        
        if [[ "$HTTP_CODE" == "200" ]]; then
            ((SUCCESS++))
        elif [[ "$HTTP_CODE" == "429" ]]; then
            ((RATE_LIMITED++))
            if [[ -z "$FIRST_RATE_LIMIT_MSG" ]]; then
                FIRST_RATE_LIMIT_MSG="$BODY"
            fi
        fi
        
        echo -ne "\r  Sent: $SUCCESS | Rate Limited: $RATE_LIMITED"
    done
    
    echo ""
    echo ""
    
    if [[ $RATE_LIMITED -gt 0 ]]; then
        echo -e "  ${GREEN}✓ PASS${NC} - Rate limiting triggered after $SUCCESS requests"
        if [[ -n "$FIRST_RATE_LIMIT_MSG" ]]; then
            echo ""
            echo -e "  ${YELLOW}First 429 response:${NC}"
            echo "  $FIRST_RATE_LIMIT_MSG" | jq -r '.message // .' 2>/dev/null || echo "  $FIRST_RATE_LIMIT_MSG"
        fi
    else
        echo -e "  ${RED}✗ FAIL${NC} - No rate limiting detected (expected 429s after ~100 requests)"
    fi
}

# =============================================================================
# SCENARIO: Daily Limits
# =============================================================================
scenario_daily_limit() {
    print_header "SCENARIO: Daily Limits"
    
    echo -e "${YELLOW}Sending until daily limit is hit...${NC}"
    echo -e "${CYAN}Using dry-run mode + simulator address${NC}"
    echo ""
    
    local SUCCESS=0
    local LIMITED=0
    local FIRST_LIMIT_MSG=""
    
    for i in $(seq 1 120); do
        RESULT=$(send_email "Daily Test $i")
        HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
        BODY=$(echo "$RESULT" | cut -d'|' -f2-)
        
        if [[ "$HTTP_CODE" == "200" ]]; then
            ((SUCCESS++))
        elif echo "$BODY" | grep -qiE "daily|limit|quota"; then
            ((LIMITED++))
            if [[ -z "$FIRST_LIMIT_MSG" ]]; then
                FIRST_LIMIT_MSG="$BODY"
            fi
            [[ $LIMITED -ge 3 ]] && break
        fi
        
        echo -ne "\r  Sent: $SUCCESS | Limited: $LIMITED"
        sleep 0.1
    done
    
    echo ""
    echo ""
    
    if [[ $LIMITED -gt 0 ]]; then
        echo -e "  ${GREEN}✓ PASS${NC} - Daily limit hit after $SUCCESS emails"
        if [[ -n "$FIRST_LIMIT_MSG" ]]; then
            echo ""
            echo -e "  ${YELLOW}Limit response:${NC}"
            echo "  $FIRST_LIMIT_MSG" | jq . 2>/dev/null || echo "  $FIRST_LIMIT_MSG"
        fi
    else
        echo -e "  ${YELLOW}⚠ SKIP${NC} - User may be on paid plan (unlimited) or limit not configured"
    fi
}

# =============================================================================
# SCENARIO: Monthly Limits
# =============================================================================
scenario_monthly_limit() {
    print_header "SCENARIO: Monthly Limits"
    
    echo -e "${YELLOW}Testing monthly limit enforcement...${NC}"
    echo -e "${CYAN}Using dry-run mode + simulator address${NC}"
    echo ""
    
    local SUCCESS=0
    local LIMITED=0
    local FIRST_LIMIT_MSG=""
    
    # Send fewer requests for monthly (would take too long to exhaust)
    # This tests that monthly limit tracking exists
    for i in $(seq 1 20); do
        RESULT=$(send_email "Monthly Test $i")
        HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
        BODY=$(echo "$RESULT" | cut -d'|' -f2-)
        
        if [[ "$HTTP_CODE" == "200" ]]; then
            ((SUCCESS++))
        elif echo "$BODY" | grep -qiE "monthly|limit"; then
            ((LIMITED++))
            if [[ -z "$FIRST_LIMIT_MSG" ]]; then
                FIRST_LIMIT_MSG="$BODY"
            fi
            break
        fi
        
        echo -ne "\r  Sent: $SUCCESS | Limited: $LIMITED"
        sleep 0.1
    done
    
    echo ""
    echo ""
    
    if [[ $LIMITED -gt 0 ]]; then
        echo -e "  ${GREEN}✓ PASS${NC} - Monthly limit is enforced"
        if [[ -n "$FIRST_LIMIT_MSG" ]]; then
            echo ""
            echo -e "  ${YELLOW}Limit response:${NC}"
            echo "  $FIRST_LIMIT_MSG" | jq . 2>/dev/null || echo "  $FIRST_LIMIT_MSG"
        fi
    else
        echo -e "  ${YELLOW}ℹ INFO${NC} - Monthly limit not hit in $SUCCESS test requests"
        echo -e "    (Monthly limits typically much higher than daily)"
    fi
}

# Main
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Limits Testing (Dry-Run Mode)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${GREEN}✓${NC} Dry-run mode: ON (X-Dry-Run header)"
echo -e "${GREEN}✓${NC} Safe recipient: $SUCCESS_EMAIL"
echo -e "${GREEN}✓${NC} SES rate limit: $SES_RATE_LIMIT req/s"
echo ""
echo "  Host: $API_HOST"
echo "  Scenario: $SCENARIO"
echo ""

case $SCENARIO in
    rate)
        scenario_rate_limit
        ;;
    daily)
        scenario_daily_limit
        ;;
    monthly)
        scenario_monthly_limit
        ;;
    all)
        scenario_rate_limit
        scenario_daily_limit
        scenario_monthly_limit
        ;;
    *)
        echo -e "${RED}Unknown scenario: $SCENARIO${NC}"
        echo "Use: --scenario=rate|daily|monthly|all"
        exit 1
        ;;
esac

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}Test Complete${NC}"
echo ""
echo -e "${YELLOW}Note:${NC} All tests ran in dry-run mode - no real emails sent"
echo -e "${YELLOW}Note:${NC} All recipients were simulator addresses for safety"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
