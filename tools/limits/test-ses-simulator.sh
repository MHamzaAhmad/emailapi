#!/bin/bash

# =============================================================================
# SES Simulator Testing
# Usage: ./test-ses-simulator.sh [bounce|complaint|success|all]
#
# Sends to AWS SES simulator addresses to test bounce/complaint handling.
# These addresses don't count against sending quotas but trigger SES events.
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# SES Simulator addresses
BOUNCE_EMAIL="bounce@simulator.amazonses.com"
COMPLAINT_EMAIL="complaint@simulator.amazonses.com"
SUCCESS_EMAIL="success@simulator.amazonses.com"

# Defaults
TEST_TYPE="${1:-all}"

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
echo -e "${BLUE}  SES Simulator Test${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Host: $API_HOST"
echo "  Test: $TEST_TYPE"
echo ""
echo -e "${YELLOW}SES Simulator Addresses:${NC}"
echo "  Bounce:    $BOUNCE_EMAIL"
echo "  Complaint: $COMPLAINT_EMAIL"
echo "  Success:   $SUCCESS_EMAIL"
echo ""

HEADERS=(-H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json")

send_email() {
    local TO_EMAIL=$1
    local TYPE=$2
    
    echo -e "${YELLOW}Sending $TYPE email to $TO_EMAIL...${NC}"
    
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "http://$API_HOST/v1.EmailService/SendEmail" \
        "${HEADERS[@]}" \
        -d "{
            \"from\": \"$FROM_EMAIL\",
            \"to\": [\"$TO_EMAIL\"],
            \"subject\": \"SES Simulator Test - $TYPE\",
            \"body\": \"Testing $TYPE scenario at $(date)\"
        }")
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        EMAIL_ID=$(echo "$BODY" | jq -r '.id // "unknown"' 2>/dev/null)
        echo -e "  ${GREEN}✓${NC} Sent! Email ID: $EMAIL_ID"
        echo -e "  ${YELLOW}→${NC} Check webhooks/SNS for $TYPE notification"
    else
        echo -e "  ${RED}✗${NC} Failed: $BODY"
    fi
    echo ""
}

case $TEST_TYPE in
    bounce)
        send_email "$BOUNCE_EMAIL" "BOUNCE"
        ;;
    complaint)
        send_email "$COMPLAINT_EMAIL" "COMPLAINT"
        ;;
    success)
        send_email "$SUCCESS_EMAIL" "SUCCESS"
        ;;
    all)
        send_email "$SUCCESS_EMAIL" "SUCCESS"
        send_email "$BOUNCE_EMAIL" "BOUNCE"
        send_email "$COMPLAINT_EMAIL" "COMPLAINT"
        ;;
    *)
        echo -e "${RED}Unknown test type: $TEST_TYPE${NC}"
        echo "Usage: $0 [bounce|complaint|success|all]"
        exit 1
        ;;
esac

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}Next Steps:${NC}"
echo "  1. Check your SNS/SQS for incoming notifications"
echo "  2. Verify reputation_worker processes bounce/complaint events"
echo "  3. Check user reputation status via admin API"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
