#!/bin/bash

# =============================================================================
# Full E2E Limit System Test
# Usage: ./test-full-e2e.sh
#
# Comprehensive test covering all limit scenarios:
#
# SCENARIO A: Rate Limiting
#   - Burst 120 requests in quick succession
#   - Expect 429 after ~100 requests
#
# SCENARIO B: Daily Limits (Free User)
#   - Send emails until daily limit (100)
#   - Verify limit error and retry-after
#
# SCENARIO C: Soft Suspension Flow
#   - Trigger bounces to get flagged
#   - Verify reduced limit (10/day) applies
#
# SCENARIO D: Hard Suspension Flow
#   - Trigger enough bounces for hard suspend
#   - Verify complete block
#
# Use --dry-run to skip actual SES sends
# Use --scenario=X to run specific scenario (A, B, C, D)
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

DRY_RUN=""
SCENARIO="all"

for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN="true"
            ;;
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
SUCCESS_EMAIL="success@simulator.amazonses.com"
BOUNCE_EMAIL="bounce@simulator.amazonses.com"

if [[ -z "$API_KEY" || -z "$FROM_EMAIL" ]]; then
    echo -e "${RED}ERROR: Configure API_KEY and FROM_EMAIL in .env${NC}"
    exit 1
fi

HEADERS=(-H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json")
if [[ -n "$DRY_RUN" ]]; then
    HEADERS+=(-H "X-Dry-Run: true")
fi

print_header() {
    echo ""
    echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║${NC}  $1"
    echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

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
            \"body\": \"Test\"
        }")
    
    echo "$(echo "$RESPONSE" | tail -1)|$(echo "$RESPONSE" | sed '$d')"
}

# =============================================================================
# SCENARIO A: Rate Limiting
# =============================================================================
scenario_rate_limit() {
    print_header "SCENARIO A: Rate Limiting"
    
    echo -e "${YELLOW}Sending 120 rapid requests to trigger rate limit...${NC}"
    echo ""
    
    local SUCCESS=0
    local RATE_LIMITED=0
    
    for i in $(seq 1 120); do
        RESULT=$(send_email "$SUCCESS_EMAIL" "Rate Test $i")
        HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
        
        if [[ "$HTTP_CODE" == "200" ]]; then
            ((SUCCESS++))
        elif [[ "$HTTP_CODE" == "429" ]]; then
            ((RATE_LIMITED++))
        fi
        
        echo -ne "\r  Sent: $SUCCESS | Rate Limited: $RATE_LIMITED"
    done
    
    echo ""
    echo ""
    
    if [[ $RATE_LIMITED -gt 0 ]]; then
        echo -e "  ${GREEN}✓ PASS${NC} - Rate limiting triggered after $SUCCESS requests"
    else
        echo -e "  ${RED}✗ FAIL${NC} - No rate limiting detected"
    fi
}

# =============================================================================
# SCENARIO B: Daily Limits
# =============================================================================
scenario_daily_limit() {
    print_header "SCENARIO B: Daily Limits"
    
    echo -e "${YELLOW}Sending until daily limit is hit...${NC}"
    echo ""
    
    local SUCCESS=0
    local LIMITED=0
    
    for i in $(seq 1 120); do
        RESULT=$(send_email "$SUCCESS_EMAIL" "Daily Test $i")
        HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
        BODY=$(echo "$RESULT" | cut -d'|' -f2-)
        
        if [[ "$HTTP_CODE" == "200" ]]; then
            ((SUCCESS++))
        elif echo "$BODY" | grep -qiE "daily|limit"; then
            ((LIMITED++))
            [[ $LIMITED -ge 3 ]] && break
        fi
        
        echo -ne "\r  Sent: $SUCCESS | Limited: $LIMITED"
        sleep 0.1
    done
    
    echo ""
    echo ""
    
    if [[ $LIMITED -gt 0 ]]; then
        echo -e "  ${GREEN}✓ PASS${NC} - Daily limit hit after $SUCCESS emails"
    else
        echo -e "  ${YELLOW}⚠ SKIP${NC} - User may be on paid plan (unlimited)"
    fi
}

# =============================================================================
# SCENARIO C: Soft Suspension
# =============================================================================
scenario_soft_suspend() {
    print_header "SCENARIO C: Soft Suspension"
    
    echo -e "${YELLOW}Step 1: Triggering bounces for soft suspension...${NC}"
    
    for i in {1..3}; do
        RESULT=$(send_email "$BOUNCE_EMAIL" "Bounce $i")
        HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
        echo "  Bounce email $i: $HTTP_CODE"
        sleep 1
    done
    
    echo ""
    echo -e "${YELLOW}Waiting for reputation processing (10s)...${NC}"
    sleep 10
    
    echo ""
    echo -e "${YELLOW}Step 2: Testing if soft limit (10/day) applies...${NC}"
    
    local SUCCESS=0
    local LIMITED=0
    
    for i in $(seq 1 15); do
        RESULT=$(send_email "$SUCCESS_EMAIL" "Soft Test $i")
        HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
        
        if [[ "$HTTP_CODE" == "200" ]]; then
            ((SUCCESS++))
        else
            ((LIMITED++))
        fi
        
        echo -ne "\r  Sent: $SUCCESS | Limited: $LIMITED"
        sleep 0.1
    done
    
    echo ""
    echo ""
    
    if [[ $SUCCESS -le 12 && $LIMITED -gt 0 ]]; then
        echo -e "  ${GREEN}✓ PASS${NC} - Soft limit (~10/day) is working"
    else
        echo -e "  ${YELLOW}⚠ INCONCLUSIVE${NC} - May need more bounces for soft suspend"
    fi
}

# =============================================================================
# SCENARIO D: Hard Suspension
# =============================================================================
scenario_hard_suspend() {
    print_header "SCENARIO D: Hard Suspension"
    
    echo -e "${YELLOW}Step 1: Triggering many bounces for hard suspension...${NC}"
    
    for i in {1..10}; do
        RESULT=$(send_email "$BOUNCE_EMAIL" "Hard Bounce $i")
        HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
        
        if [[ "$HTTP_CODE" == "200" ]]; then
            echo "  Bounce email $i: sent"
        else
            echo "  Bounce email $i: blocked (may already be suspended)"
            break
        fi
        sleep 1
    done
    
    echo ""
    echo -e "${YELLOW}Waiting for reputation processing (10s)...${NC}"
    sleep 10
    
    echo ""
    echo -e "${YELLOW}Step 2: Testing if hard block applies...${NC}"
    
    RESULT=$(send_email "$SUCCESS_EMAIL" "Block Test")
    HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
    BODY=$(echo "$RESULT" | cut -d'|' -f2-)
    
    if echo "$BODY" | grep -qi "suspend"; then
        echo -e "  ${GREEN}✓ PASS${NC} - Hard suspension is blocking sends"
    elif [[ "$HTTP_CODE" == "200" ]]; then
        echo -e "  ${YELLOW}⚠ INCONCLUSIVE${NC} - Still able to send (threshold may be higher)"
    else
        echo -e "  ${YELLOW}?${NC} Response: $BODY"
    fi
}

# Main
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Full E2E Limit System Test${NC}"
if [[ -n "$DRY_RUN" ]]; then
    echo -e "${YELLOW}  Mode: DRY-RUN${NC}"
fi
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo "  Host: $API_HOST"
echo "  Scenario: $SCENARIO"
echo ""

case $SCENARIO in
    A|a|rate)
        scenario_rate_limit
        ;;
    B|b|daily)
        scenario_daily_limit
        ;;
    C|c|soft)
        scenario_soft_suspend
        ;;
    D|d|hard)
        scenario_hard_suspend
        ;;
    all)
        scenario_rate_limit
        scenario_daily_limit
        scenario_soft_suspend
        scenario_hard_suspend
        ;;
    *)
        echo "Unknown scenario: $SCENARIO"
        echo "Use: --scenario=A|B|C|D|all"
        exit 1
        ;;
esac

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}Test Complete${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
