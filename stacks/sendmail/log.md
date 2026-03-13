# Sendmail

## Status: SKIPPED

## Research
- Project website: https://www.proofpoint.com/ — now owned by Proofpoint, no Docker support
- Docker Hub search: only small niche images found (nicescale/sendmail tied to dnspod.cn, others undocumented)
- No official or well-maintained general-purpose Sendmail Docker image exists

## Reason for Skip
Sendmail is a legacy Unix MTA now owned by Proofpoint. No official Docker image exists. The community images are either vendor-locked (dnspod.cn), undocumented, or unmaintained. Most users have migrated to Postfix or other modern MTAs.
