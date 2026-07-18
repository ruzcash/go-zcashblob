# Security Policy

## Supported versions

Security fixes are provided for the latest patch release in the latest `0.x`
minor series.

| Version | Supported |
| --- | --- |
| 0.1.x | Yes |
| Earlier versions | No |

## Reporting a vulnerability

Please use GitHub's
[private vulnerability reporting](https://github.com/ruzcash/go-zcashblob/security/advisories/new)
for this repository. Do not open a public issue or pull request for an
unpatched vulnerability.

Include the affected version, impact, reproduction steps, and any suggested
mitigation. Remove secrets and unrelated private transaction data. The report
will be acknowledged through the private advisory and coordinated there until
a fix and disclosure plan are ready.

## Security scope

This library parses untrusted binary data but does not perform Zcash consensus
validation. Applications must use a consensus node when validity matters.
