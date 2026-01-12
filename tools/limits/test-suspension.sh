#!/bin/bash

# =============================================================================
# Suspension Testing
# Usage: ./test-suspension.sh [--dry-run]
#
# Tests that suspended users are blocked from sending emails.
# Requires you to manually suspend the test user first via admin API.
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

DRY_RUN=""
for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN="true"
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
echo -e "${BLUE}  Suspension Test${NC}"
if [[ -n "$DRY_RUN" ]]; then
    echo -e "${YELLOW}  Mode: DRY-RUN${NC}"
fi
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Build headers
HEADERS=(-H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json")
if [[ -n "$DRY_RUN" ]]; then
    HEADERS+=(-H "X-Dry-Run: true")
fi

echo -e "${YELLOW}Prerequisites:${NC}"
echo "  1. Suspend your test user via admin API first:"
echo ""
echo -e "     ${CYAN}# Hard suspend (blocks all sends):${NC}"
echo "     curl -X POST http://$API_HOST/admin.ReputationService/SuspendUser \\"
echo "       -H 'Authorization: Bearer ADMIN_KEY' \\"
echo "       -H 'Content-Type: application/json' \\"
echo "       -d '{\"user_id\": \"USER_ID\", \"reason\": \"Testing\"}'"
echo ""
echo -e "     ${CYAN}# Soft suspend (reduced limits):${NC}"
echo "     curl -X POST http://$API_HOST/admin.ReputationService/SoftSuspendUser \\"
echo "       -H 'Authorization: Bearer ADMIN_KEY' \\"
echo "       -H 'Content-Type: application/json' \\"
echo "       -d '{\"user_id\": \"USER_ID\", \"reason\": \"Testing\"}'"
echo ""

read -p "Press Enter to continue with send test, or Ctrl+C to cancel..."

echo ""
echo -e "${YELLOW}Attempting to send email...${NC}"

RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "http://$API_HOST/v1.EmailService/SendEmail" \
    "${HEADERS[@]}" \
    -d "{
        \"from\": \"$FROM_EMAIL\",
        \"to\": [\"success@simulator.amazonses.com\"],
        \"subject\": \"Suspension Test\",
        \"body\": \"Testing suspension\"
    }")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo ""
echo -e "${BLUE}Response:${NC}"
echo "  HTTP Code: $HTTP_CODE"
echo "  Body: $BODY" | jq . 2>/dev/null || echo "  $BODY"
echo ""

# Check for suspension error
if echo "$BODY" | grep -qi "suspend"; then
    echo -e "${GREEN}✓ Suspension is working - request was blocked!${NC}"
elif [[ "$HTTP_CODE" == "200" ]]; then
    echo -e "${YELLOW}⚠ Request succeeded - user may not be suspended${NC}"
else
    echo -e "${YELLOW}Request failed with different error${NC}"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}To unsuspend:${NC}"
echo "  curl -X POST http://$API_HOST/admin.ReputationService/UnsuspendUser \\"
echo "    -H 'Authorization: Bearer ADMIN_KEY' \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"user_id\": \"USER_ID\"}'"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
