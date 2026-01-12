# Limits Testing

Test scripts for verifying rate limits, usage limits, and suspension logic against localhost or remote APIs.

## Prerequisites

- `curl` and `jq` installed
- API key with appropriate permissions
- Redis running (for limits to work)

## Quick Start

```bash
# Copy and configure .env
cp .env.example .env
# Edit .env with your API key and settings

# Run full e2e test suite
./test-full-e2e.sh --dry-run

# Or run individual scenarios
./test-full-e2e.sh --scenario=A  # Rate limit
./test-full-e2e.sh --scenario=B  # Daily limit
./test-full-e2e.sh --scenario=C  # Soft suspend
./test-full-e2e.sh --scenario=D  # Hard suspend
```

## Test Scripts

| Script | Purpose |
|--------|---------|
| `test-full-e2e.sh` | **Comprehensive test** - all scenarios in sequence |
| `test-e2e-suspension.sh` | E2E flow: baseline → bounces → soft → hard suspend |
| `test-e2e-daily-limit.sh` | E2E flow: send until daily limit hit |
| `test-rate-limit.sh` | Burst requests to trigger 429s |
| `test-daily-limit.sh` | Exhaust daily quota (simple) |
| `test-ses-simulator.sh` | Send to SES simulator for bounce/complaint |
| `test-suspension.sh` | Test suspended account blocking |

## E2E Test Scenarios

### Scenario A: Rate Limiting
Sends 120 rapid requests, expects 429 after ~100.

### Scenario B: Daily Limits
Sends until daily limit (100 for free), verifies error.

### Scenario C: Soft Suspension
1. Trigger bounces (3x)
2. Wait for reputation processing
3. Verify reduced limit (10/day) applies

### Scenario D: Hard Suspension
1. Trigger many bounces (10x)
2. Wait for reputation processing
3. Verify complete block

## SES Simulator Addresses

AWS SES provides simulator addresses for testing:

| Address | Result |
|---------|--------|
| `success@simulator.amazonses.com` | Successful delivery |
| `bounce@simulator.amazonses.com` | Hard bounce |
| `complaint@simulator.amazonses.com` | Spam complaint |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `API_HOST` | API host (default: localhost:8080) |
| `API_KEY` | Your API key |
| `FROM_EMAIL` | Verified sender email |

