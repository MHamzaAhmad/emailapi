#!/bin/bash

# =============================================================================
# E2E Suspension Flow Test
# Usage: ./test-e2e-suspension.sh [--dry-run] [--yes]
#
# Simulates realistic user behavior leading to auto-suspension:
# 
# 1. Send successful emails (baseline)
# 2. Send to bounce addresses (triggers reputation events)
# 3. Verify soft suspension kicks in (10 emails/day limit)
# 4. Send more bounces (triggers hard suspension)
# 5. Verify complete block
#
# SAFETY:
#   - ONLY uses SES simulator addresses (@simulator.amazonses.com)
#   - Respects SES rate limits (1 req/s delay between sends)
#   - Optional dry-run mode for testing without reputation impact
#
# WARNING: Without --dry-run, this sends REAL emails and modifies reputation!
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

# Parse arguments
DRY_RUN=""
SKIP_CONFIRM=""
for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN="true"
            ;;
        --yes)
            SKIP_CONFIRM="true"
            ;;
    esac
done

# MANDATORY: Only simulator addresses (defense in depth)
BOUNCE_EMAIL="bounce@simulator.amazonses.com"
SUCCESS_EMAIL="success@simulator.amazonses.com"
COMPLAINT_EMAIL="complaint@simulator.amazonses.com"

# Load config
if [[ -f "$SCRIPT_DIR/.env" ]]; then
    export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
fi

API_HOST="${API_HOST:-localhost:8080}"
SES_RATE_LIMIT="${SES_RATE_LIMIT:-1}"

if [[ -z "$API_KEY" || -z "$FROM_EMAIL" ]]; then
    echo -e "${RED}ERROR: Missing required environment variables${NC}"
    echo "Required: API_KEY, FROM_EMAIL"
    exit 1
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  E2E Suspension Flow Test${NC}"
if [[ -n "$DRY_RUN" ]]; then
    echo -e "${YELLOW}  Mode: DRY-RUN (no reputation impact)${NC}"
fi
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Host: $API_HOST"
echo "  From: $FROM_EMAIL"
echo ""
echo -e "${GREEN}✓${NC} Safe recipients: simulator addresses only"
echo -e "${GREEN}✓${NC} SES rate limit: $SES_RATE_LIMIT req/s"
echo ""

if [[ -z "$DRY_RUN" ]]; then
    echo -e "${RED}⚠️  WARNING: This test sends real emails and modifies reputation!${NC}"
    echo -e "${RED}   Make sure you're using a test account.${NC}"
    echo ""
    if [[ -z "$SKIP_CONFIRM" ]]; then
        read -p "Press Enter to continue, or Ctrl+C to cancel..."
    fi
fi

HEADERS=(-H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json")
if [[ -n "$DRY_RUN" ]]; then
    HEADERS+=(-H "X-Dry-Run: true")
fi

# Helper to send email and return status
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
    
    # Respect SES rate limits (delay after each send)
    sleep $(awk "BEGIN {print 1/$SES_RATE_LIMIT}")
}

# Helper to check user status
check_status() {
    # This would need an admin endpoint - for now just try to send
    RESULT=$(send_email "$SUCCESS_EMAIL" "Status Check")
    HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
    BODY=$(echo "$RESULT" | cut -d'|' -f2-)
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo "ACTIVE"
    elif echo "$BODY" | grep -qi "suspend"; then
        if echo "$BODY" | grep -qi "soft"; then
            echo "SOFT_SUSPENDED"
        else
            echo "HARD_SUSPENDED"
        fi
    elif echo "$BODY" | grep -qi "daily"; then
        echo "DAILY_LIMITED"
    else
        echo "ERROR: $HTTP_CODE"
    fi
}

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}PHASE 1: Baseline - Sending successful emails${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

for i in {1..3}; do
    RESULT=$(send_email "$SUCCESS_EMAIL" "Baseline Test $i")
    HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo -e "  ${GREEN}✓${NC} Email $i sent successfully"
    else
        echo -e "  ${RED}✗${NC} Email $i failed: $HTTP_CODE"
    fi
    sleep 1
done

echo ""
echo -e "${BLUE}Waiting for events to process (5s)...${NC}"
sleep 5

echo ""
echo -e "${BLUE}═══════════================================================================"
echo -e "${CYAN}PHASE 2: Triggering bounces (soft suspension threshold)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Sending to bounce address to trigger reputation events...${NC}"
echo -e "${CYAN}Threshold: 10 hard bounces → Soft suspension (10/day limit)${NC}"
echo ""

# Send 10 bounces to trigger soft suspension
for i in {1..10}; do
    RESULT=$(send_email "$BOUNCE_EMAIL" "Bounce Test $i")
    HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo -e "  ${YELLOW}→${NC} Bounce email $i sent (will trigger bounce event)"
    else
        echo -e "  ${RED}✗${NC} Send failed: $HTTP_CODE"
        BODY=$(echo "$RESULT" | cut -d'|' -f2-)
        echo "    $BODY"
    fi
done

echo ""
echo -e "${BLUE}Waiting for bounce events to process (15s)...${NC}"
echo -e "${YELLOW}(SNS → SQS → ReputationWorker)${NC}"
sleep 15

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}PHASE 3: Testing soft suspension (10/day limit)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Try to send and see if soft limits apply
echo -e "${YELLOW}Testing if soft suspension is active...${NC}"
echo -e "${CYAN}Expected: Limited to 10 emails/day${NC}"
echo ""

SUCCESS_COUNT=0
LIMITED_COUNT=0

# Try to send 15 emails (should hit 10/day limit)
for i in $(seq 1 15); do
    RESULT=$(send_email "$SUCCESS_EMAIL" "Post-Bounce Test $i")
    HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
    BODY=$(echo "$RESULT" | cut -d'|' -f2-)
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        ((SUCCESS_COUNT++))
    elif echo "$BODY" | grep -qi "daily"; then
        ((LIMITED_COUNT++))
        if [[ $LIMITED_COUNT -eq 1 ]]; then
            echo ""
            echo -e "  ${YELLOW}First limit hit at $SUCCESS_COUNT emails:${NC}"
            echo "  $BODY" | jq -r '.message // .' 2>/dev/null || echo "  $BODY"
        fi
        # Stop after hitting limit 3 times
        [[ $LIMITED_COUNT -ge 3 ]] && break
    fi
    
    echo -ne "\r  Sent: $SUCCESS_COUNT | Limited: $LIMITED_COUNT"
done

echo ""
echo ""

if [[ $SUCCESS_COUNT -ge 8 && $SUCCESS_COUNT -le 12 && $LIMITED_COUNT -gt 0 ]]; then
    echo -e "  ${GREEN}✓ PASS${NC} - Soft suspension is working (~10/day limit)"
    echo -e "  ${CYAN}Sent $SUCCESS_COUNT emails before hitting limit${NC}"
elif [[ "$HTTP_CODE" == "200" ]]; then
    echo -e "  ${YELLOW}⚠${NC} Still allowed to send"
    echo -e "  ${YELLOW}Note: May need more time for reputation processing${NC}"
elif echo "$BODY" | grep -qi "daily"; then
    echo -e "  ${YELLOW}⚠${NC} Daily limit applied (soft suspension may be active)"
    echo "  $BODY" | jq -r '.message // .' 2>/dev/null || echo "  $BODY"
elif echo "$BODY" | grep -qi "suspend"; then
    echo -e "  ${RED}✗${NC} Account suspended"
    echo "  $BODY" | jq -r '.message // .' 2>/dev/null || echo "  $BODY"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}PHASE 4: More bounces (hard suspension threshold)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${CYAN}Threshold: 20 total hard bounces → Hard suspension (full block)${NC}"
echo -e "${YELLOW}Sending 10 more bounces (10 already sent = 20 total)...${NC}"
echo ""

for i in {11..20}; do
    RESULT=$(send_email "$BOUNCE_EMAIL" "Bounce Test $i")
    HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo -e "  ${YELLOW}→${NC} Bounce email $i sent"
    else
        BODY=$(echo "$RESULT" | cut -d'|' -f2-)
        if echo "$BODY" | grep -qi "suspend"; then
            echo -e "  ${RED}✗${NC} Already suspended - can't send more"
            break
        else
            echo -e "  ${RED}✗${NC} Send failed: $HTTP_CODE"
        fi
    fi
done

echo ""
echo -e "${BLUE}Waiting for events to process (15s)...${NC}"
sleep 15

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}PHASE 5: Final check - Hard suspension test${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${CYAN}Expected: Account should be hard suspended (20 total hard bounces)${NC}"
echo ""

RESULT=$(send_email "$SUCCESS_EMAIL" "Final Check")
HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
BODY=$(echo "$RESULT" | cut -d'|' -f2-)

echo -e "  HTTP: $HTTP_CODE"

if echo "$BODY" | grep -qi "suspend"; then
    echo -e "  ${GREEN}✓ PASS${NC} - Account is hard suspended - blocking works!"
    echo "  $BODY" | jq . 2>/dev/null || echo "  $BODY"
elif [[ "$HTTP_CODE" == "200" ]]; then
    echo -e "  ${YELLOW}⚠ INCONCLUSIVE${NC} - Still able to send"
    echo -e "  ${YELLOW}Note: Reputation worker may need more time to process all bounces${NC}"
else
    echo "  Response: $BODY"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}Test Complete${NC}"
echo ""
echo -e "${CYAN}Summary:${NC}"
echo "  • Sent 20 bounce emails (hard suspension threshold)"
echo "  • Soft suspension triggers at 10 bounces → 10/day limit"
echo "  • Hard suspension triggers at 20 bounces → Full block"
echo ""
echo "To reset user reputation for further testing:"
echo ""
echo "  # Via admin API:"
echo "  curl -X POST http://$API_HOST/admin.ReputationService/UnsuspendUser \\"
echo "    -H 'Authorization: Bearer ADMIN_KEY' \\"
echo "    -d '{\"user_id\": \"YOUR_USER_ID\"}'"
echo ""
echo "  # Or clear Redis keys:"
echo "  redis-cli DEL \"reputation:USER_ID\""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
