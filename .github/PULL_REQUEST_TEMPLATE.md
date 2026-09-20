## Description

Please include a summary of the changes and the related issue. Specify the motivation and context.

Fixes # (issue)

## Type of Change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature / adapter (non-breaking change adding new diagnostic capabilities)
- [ ] Breaking change (fix or feature that would cause existing behavior to change)
- [ ] Documentation update
- [ ] Performance / Refactoring

## How Has This Been Tested?

Please describe the tests you ran to verify your changes. Include commands used and output snippets if applicable:

```bash
# Example test run
go test -v -race ./...
./bin/why <command> <target>
```

- [ ] Unit tests added / updated
- [ ] Tested with live target
- [ ] Tested `--json` schema output
- [ ] Verified race condition checks (`go test -race`)

## Diagnostic Contract Compliance

- [ ] Adapter produces structured facts in `model.Check.Evidence`
- [ ] Adapter **does not** format or print presentation text directly
- [ ] Cause engine produces deterministic `model.Cause` with confidence and actionable remediation

## Checklist

- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code in hard-to-understand areas
- [ ] I have updated corresponding documentation in `docs/` or `README.md`
- [ ] My changes generate no new warnings or lint errors
