# Security Policy

## Supported Versions

We provide security updates and patches for the following versions of `why`:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

## Reporting a Vulnerability

We take the security of `why` seriously. If you discover a vulnerability or security flaw in `why` (such as credential leakage during SSH or HTTP header inspection, arbitrary code execution, or network amplification risks), please follow these steps:

1. **Do not create a public GitHub issue.**
2. Send an email with details and reproduction steps to: **kavix@yahoo.com**
3. Include:
   - Target protocol and flags used
   - Operating system and Go version
   - A Minimal Reproducible Example (MRE)
   - Any potential mitigation or patch ideas

## Response Timeline

- **Initial Response:** Within 48 hours of notification.
- **Triage & Status Update:** Within 5 business days.
- **Fix & Advisory:** A patch release and security advisory will be coordinated before public disclosure.
