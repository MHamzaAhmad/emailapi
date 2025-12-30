# Reputation System

This document describes the email sender reputation system, including thresholds, automated actions, and administrative processes.

## Overview

The reputation system tracks bounce and complaint incidents per user to:
1. **Identify problematic senders** before they damage platform deliverability
2. **Reduce rate limits** for flagged accounts while under investigation
3. **Auto-suspend** severe violators (e.g., spammers with purchased lists)

---

## Incident Tracking

Every bounce and complaint is recorded with:
- Incident type: `bounce_hard`, `bounce_soft`, `complaint`
- Message ID and recipient hash
- SES-provided diagnostic codes

### Scoring Weights

| Incident Type | Points |
|--------------|--------|
| Hard Bounce | 5 |
| Soft Bounce | 1 |
| Complaint | 10 |

The suspension score is calculated as:
```
lifetime_score = (hard_bounces × 5) + soft_bounces + (complaints × 10)
recent_score = (bounces_30d × 3) + (complaints_30d × 10)
final_score = (lifetime_score × 0.3) + (recent_score × 0.7)
```

---

## Thresholds and Actions

### Flagging (Manual Review Required)

User is **flagged** when ANY of these conditions are met:

| Condition | Description |
|-----------|-------------|
| Score > 50 | High combined reputation score |
| 3+ complaints in 30 days | Pattern of spam complaints |
| 10+ hard bounces total | Sending to many invalid addresses |

**Effect**: Rate limit reduced to **10%** of normal. User can still send but at reduced capacity.

### Auto-Suspension (Cannot Send)

User is **auto-suspended** when ANY of these conditions are met:

| Condition | Description |
|-----------|-------------|
| 5+ complaints in 30 days | Severe spam behavior |
| 20+ hard bounces total | Likely using purchased/invalid list |
| Score > 100 | Critical reputation score |

**Effect**: All email sending is **blocked**. User must contact support.

---

## Rate Limits by Status

| Status | Rate Limit | Can Send? |
|--------|------------|-----------|
| Normal | 100% | ✅ Yes |
| Flagged | 10% | ✅ Yes (reduced) |
| Suspended | 0% | ❌ No |

---

## Recovery Path

1. **Flagged users** can continue sending at reduced capacity
2. **Suspended users** must contact support
3. Admin reviews incidents and decides action:
   - Clear bad addresses from their list
   - Enable double opt-in
   - Improve list hygiene
4. After fixes verified, admin can unsuspend

---
