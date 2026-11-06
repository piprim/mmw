# Security Policy

## Supported versions

Security fixes are applied to the latest release and the `main` branch only. Older versions are not patched; please upgrade.

## Reporting a vulnerability

**Do not open a public issue, pull request, or discussion for a security problem.**

Report it privately through GitHub: [Report a vulnerability](https://github.com/piprim/mmw/security/advisories/new) (Security tab → Advisories → Report a vulnerability).

Please include:

- the affected package (`pkg/platform/...`, `cmd/mmw`) and version or commit,
- a description of the issue and its impact,
- steps to reproduce or a proof of concept,
- a suggested fix, if you have one.

## What to expect

- Acknowledgement within 7 days.
- An assessment and, if the report is confirmed, a fix plan as soon as practical; you will be kept informed along the way.
- Coordinated disclosure: the advisory is published once a fix is released, with credit to you unless you prefer to stay anonymous.

## Scope

In scope: code in this repository, namely the runtime platform (`pkg/platform`) and the `mmw` CLI (`cmd/mmw`).

Out of scope: vulnerabilities in third-party dependencies (report those upstream; Dependabot tracks them here) and issues in applications built on `mmw` that stem from their own code or configuration.
