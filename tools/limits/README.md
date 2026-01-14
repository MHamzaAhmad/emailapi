# Limits Testing

Test scripts for verifying rate limits, usage limits, and suspension logic against localhost or remote APIs.

## Prerequisites

- `curl` and `jq` installed
- API key with appropriate permissions
- Redis running (for limits to work)
- Verified sender email in SES

## Quick Start

```bash
# Copy and configure .env
cp .env.example .env
# Edit .env with your API key and FROM_EMAIL

# Test limits in dry-run mode (SAFE - no real emails)
./test-limits-dry-run.sh

# Test suspension flow (CAUTION - modifies reputation)
./test-e2e-suspension.sh
```

## Test Scripts

| Script | Purpose | Sends Real Emails | Modifies Reputation |
|--------|---------|-------------------|---------------------|
| `test-limits-dry-run.sh` | Test rate, daily, monthly limits | **NO** (dry-run) | **NO** |
| `test-e2e-suspension.sh` | Test soft/hard suspension | YES (simulator only) | **YES** (unless --dry-run) |

## Dry-Run Limits Testing

Tests rate limiting, daily limits, and monthly limits **WITHOUT sending real emails**:

```bash
# Run all limit scenarios
./test-limits-dry-run.sh

# Run specific scenario
./test-limits-dry-run.sh --scenario=rate    # Rate limiting only
./test-limits-dry-run.sh --scenario=daily   # Daily limit only
./test-limits-dry-run.sh --scenario=monthly # Monthly limit only
```

**What it tests**:
- **Rate Limit**: Sends 120 rapid requests, expects 429 after ~100
- **Daily Limit**: Sends until daily quota exhausted (100 for free tier)
- **Monthly Limit**: Tests monthly limit enforcement (if configured)

**Safety Features**:
- ✅ Uses `X-Dry-Run` header (no real SES API calls)
- ✅ Only sends to `success@simulator.amazonses.com` (defense in depth)
- ✅ Respects SES rate limits (configurable delay)

**Note**: Even in dry-run mode, we use simulator addresses. If the dry-run header fails for any reason, we still only send to safe AWS simulator addresses.

## E2E Suspension Testing

Tests the complete suspension flow using SES simulator addresses:

```bash
# Run with confirmation prompt (sends real emails)
./test-e2e-suspension.sh

# Run in dry-run mode (no reputation impact)
./test-e2e-suspension.sh --dry-run

# Skip confirmation (CI/CD)
./test-e2e-suspension.sh --yes
```

**What it tests**:
1. **Baseline**: Send to `success@simulator.amazonses.com` (3 emails) ✅
2. **Soft Suspension**: Send to `bounce@simulator.amazonses.com` (10 bounces)
   - Triggers soft suspension at **10 hard bounces**
   - Account remains active but daily limit reduced to **10 emails/day**
   - Can still send but with restrictions
3. **Test 10/day Limit**: Send 15 emails to verify soft suspension limit
   - Should succeed for first ~10 emails
   - Should hit daily limit and block further sends
4. **Hard Suspension**: Send more bounces (10 additional for 20 total)
   - Triggers hard suspension at **20 hard bounces**
   - All sends completely blocked with suspension error
5. **Verification**: Confirm account is fully blocked

**⚠️ Warning**: 
- Without `--dry-run`, this sends **real emails** (to simulator addresses only)
- **Modifies your reputation score** in Redis
- Use with test accounts or be prepared to reset reputation

**Safety Features**:
- ✅ Only uses SES simulator addresses (`@simulator.amazonses.com`)
- ✅ Respects SES rate limits (1 req/s delay)
- ✅ Optional dry-run mode: `--dry-run`
- ✅ Confirmation prompt (skip with `--yes`)

## AWS SES Simulator Addresses

AWS provides special addresses for testing without affecting your reputation or quota:

| Address | Result | Use Case |
|---------|--------|----------|
| `success@simulator.amazonses.com` | ✅ Successful delivery | Testing happy path, limits |
| `bounce@simulator.amazonses.com` | ⚠️ Hard bounce | Testing bounce handling, soft/hard suspension |
| `complaint@simulator.amazonses.com` | 🚫 Spam complaint | Testing complaint handling, suspension |

**Important**: These addresses:
- Don't count against your SES sending quota
- Trigger real SES events (bounces, complaints) via SNS/SQS
- Are **the only addresses** used by our test scripts (defense in depth)

[AWS SES Simulator Documentation →](https://docs.aws.amazon.com/ses/latest/dg/send-an-email-from-console.html#send-email-simulator)

## SES Rate Limiting

All scripts automatically respect SES rate limits to avoid throttling:

```bash
# In .env
SES_RATE_LIMIT=1  # Requests per second (default: 1)
```

- **Sandbox accounts**: 1 req/s recommended
- **Production accounts**: 5-10 req/s (based on your quota)
- Scripts add delays between requests: `1/SES_RATE_LIMIT` seconds

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `API_HOST` | API host | `localhost:8080` | No |
| `API_KEY` | Your API key | - | **Yes** |
| `FROM_EMAIL` | Verified sender email | - | **Yes** |
| `SUCCESS_EMAIL` | Success simulator | `success@simulator.amazonses.com` | No |
| `BOUNCE_EMAIL` | Bounce simulator | `bounce@simulator.amazonses.com` | No |
| `COMPLAINT_EMAIL` | Complaint simulator | `complaint@simulator.amazonses.com` | No |
| `SES_RATE_LIMIT` | Requests per second | `1` | No |

## Troubleshooting

### Daily limit already exhausted

Reset your daily counter in Redis:

```bash
redis-cli DEL "limit:daily:YOUR_USER_ID"
```

### Account suspended

Reset your reputation score:

```bash
# Clear reputation data
redis-cli DEL "reputation:YOUR_USER_ID"

# Or use admin API
curl -X POST http://localhost:8080/admin.ReputationService/UnsuspendUser \
  -H 'Authorization: Bearer ADMIN_KEY' \
  -H 'Content-Type: application/json' \
  -d '{"user_id": "YOUR_USER_ID"}'
```

### SES Throttling (429 from AWS)

Increase delays between requests:

```bash
# In .env
SES_RATE_LIMIT=0.5  # 1 request every 2 seconds
```

### Dry-run not working

Verify your API respects the `X-Dry-Run` header:

```bash
# Test manually
curl -X POST http://localhost:8080/v1.EmailService/SendEmail \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "X-Dry-Run: true" \
  -d '{
    "from": "test@yourdomain.com",
    "to": ["success@simulator.amazonses.com"],
    "subject": "Dry-run test",
    "body": "This should not send"
  }'

# Response should contain: "[DRY-RUN]"
```

## Safety Checklist

Before running any tests:

- [ ] `.env` is configured with your API key
- [ ] `FROM_EMAIL` is verified in SES
- [ ] You're using a test account (or prepared to reset)
- [ ] Redis is running
- [ ] You understand which tests modify reputation

For dry-run limits test:
- [ ] Script will use `X-Dry-Run` header ✅
- [ ] Only simulator addresses as recipients ✅
- [ ] No reputation impact ✅

For E2E suspension test:
- [ ] You've reviewed what the test does
- [ ] You're prepared to reset reputation if needed
- [ ] Consider using `--dry-run` flag first

## Removed Scripts

The following scripts were removed in favor of the consolidated approach:

- `test-full-e2e.sh` → Use `test-limits-dry-run.sh` + `test-e2e-suspension.sh`
- `test-rate-limit.sh` → Use `test-limits-dry-run.sh --scenario=rate`
- `test-daily-limit.sh` → Use `test-limits-dry-run.sh --scenario=daily`
- `test-e2e-daily-limit.sh` → Use `test-limits-dry-run.sh --scenario=daily`
- `test-ses-simulator.sh` → Functionality integrated into main tests
- `test-suspension.sh` → Use `test-e2e-suspension.sh`
