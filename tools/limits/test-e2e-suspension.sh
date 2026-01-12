#!/bin/bash

# =============================================================================
# E2E Suspension Flow Test
# Usage: ./test-e2e-suspension.sh
#
# Simulates realistic user behavior leading to auto-suspension:
# 
# 1. Send successful emails (baseline)
# 2. Send to bounce addresses (triggers reputation events)
# 3. Verify soft suspension kicks in (10 emails/day limit)
# 4. Send more bounces (triggers hard suspension)
# 5. Verify complete block
#
# This is a REAL test - it sends actual emails and modifies reputation!
# Use with a test account or reset afterwards.
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

# SES Simulator addresses
BOUNCE_EMAIL="bounce@simulator.amazonses.com"
SUCCESS_EMAIL="success@simulator.amazonses.com"

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
echo -e "${BLUE}  E2E Suspension Flow Test${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Host: $API_HOST"
echo "  From: $FROM_EMAIL"
echo ""
echo -e "${RED}⚠️  WARNING: This test sends real emails and modifies reputation!${NC}"
echo -e "${RED}   Make sure you're using a test account.${NC}"
echo ""
read -p "Press Enter to continue, or Ctrl+C to cancel..."

HEADERS=(-H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json")

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
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}PHASE 2: Triggering bounces (soft suspension threshold)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Sending to bounce address to trigger reputation events...${NC}"
echo ""

for i in {1..3}; do
    RESULT=$(send_email "$BOUNCE_EMAIL" "Bounce Test $i")
    HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo -e "  ${YELLOW}→${NC} Bounce email $i sent (will trigger bounce event)"
    else
        echo -e "  ${RED}✗${NC} Send failed: $HTTP_CODE"
        BODY=$(echo "$RESULT" | cut -d'|' -f2-)
        echo "    $BODY"
    fi
    sleep 1
done

echo ""
echo -e "${BLUE}Waiting for bounce events to process (10s)...${NC}"
echo -e "${YELLOW}(SNS → SQS → ReputationWorker)${NC}"
sleep 10

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}PHASE 3: Testing after bounces${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Try to send and see if soft limits apply
echo -e "${YELLOW}Testing send capability...${NC}"
RESULT=$(send_email "$SUCCESS_EMAIL" "Post-Bounce Test")
HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
BODY=$(echo "$RESULT" | cut -d'|' -f2-)

echo ""
echo -e "  HTTP: $HTTP_CODE"
if [[ "$HTTP_CODE" == "200" ]]; then
    echo -e "  ${GREEN}✓${NC} Still allowed to send"
    echo -e "  ${YELLOW}Note: May have reduced daily limit if soft suspended${NC}"
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

for i in {4..7}; do
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
    sleep 1
done

echo ""
echo -e "${BLUE}Waiting for events to process (10s)...${NC}"
sleep 10

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}PHASE 5: Final check - should be blocked${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

RESULT=$(send_email "$SUCCESS_EMAIL" "Final Check")
HTTP_CODE=$(echo "$RESULT" | cut -d'|' -f1)
BODY=$(echo "$RESULT" | cut -d'|' -f2-)

echo -e "  HTTP: $HTTP_CODE"

if echo "$BODY" | grep -qi "suspend"; then
    echo -e "  ${GREEN}✓${NC} Account is suspended - blocking works!"
    echo "  $BODY" | jq . 2>/dev/null || echo "  $BODY"
elif [[ "$HTTP_CODE" == "200" ]]; then
    echo -e "  ${YELLOW}⚠${NC} Still able to send - suspension threshold may not be reached"
else
    echo "  Response: $BODY"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}Test Complete${NC}"
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
