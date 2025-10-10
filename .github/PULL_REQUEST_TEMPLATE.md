## Description

<!-- Provide a clear and concise description of the changes in this PR -->

## Type of Change

<!-- Mark the relevant option with an "x" -->

- [ ] 🐛 Bug fix (non-breaking change that fixes an issue)
- [ ] ✨ New feature (non-breaking change that adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📝 Documentation update
- [ ] 🔧 Configuration change
- [ ] ♻️ Refactoring (no functional changes)
- [ ] ⚡ Performance improvement
- [ ] 🔒 Security fix
- [ ] 🎨 UI/UX improvement
- [ ] ✅ Test coverage improvement

## Related Issues

<!-- Link related issues using keywords: Fixes #123, Closes #456, Related to #789 -->

Fixes #
Related to #

## Changes Made

<!-- List the specific changes made in this PR -->

-
-
-

## Testing

<!-- Describe the testing you've done -->

### Test Environment

- OS:
- Go version:
- Browser (if UI changes):

### Test Cases

<!-- Describe what was tested and how -->

- [ ] Unit tests pass (`go test ./...`)
- [ ] Integration tests pass (if applicable)
- [ ] Manual testing completed
- [ ] Tested with different LDAP servers (if applicable):
  - [ ] Active Directory
  - [ ] OpenLDAP
  - [ ] FreeIPA
  - [ ] Other: \***\*\_\_\_\*\***

### Test Results

<!-- Paste relevant test output or screenshots -->

```
<!-- Test output here -->
```

## Code Quality Checklist

### Code Standards

- [ ] Code follows the project's style guidelines
- [ ] golangci-lint passes (`golangci-lint run`)
- [ ] ESLint passes (if frontend changes) (`pnpm lint`)
- [ ] Prettier formatting applied (`pnpm prettier --write .`)
- [ ] TypeScript type checks pass (if applicable) (`pnpm js:build`)
- [ ] No new compiler warnings
- [ ] No `TODO` comments for core functionality

### Testing & Coverage

- [ ] New tests added for new functionality
- [ ] Existing tests updated for changed functionality
- [ ] Test coverage maintained or improved
- [ ] Edge cases covered
- [ ] Error handling tested

### Documentation

- [ ] Code comments added/updated for complex logic
- [ ] Public APIs documented (if applicable)
- [ ] README updated (if needed)
- [ ] Configuration docs updated (if config changes)
- [ ] Migration guide provided (if breaking changes)

### Security

- [ ] No sensitive data in code or logs
- [ ] Input validation implemented
- [ ] SQL/LDAP injection prevention verified
- [ ] Authentication/authorization checked
- [ ] Dependencies scanned for vulnerabilities
- [ ] Security best practices followed

### Performance

- [ ] No obvious performance regressions
- [ ] Database queries optimized (if applicable)
- [ ] LDAP queries efficient (if applicable)
- [ ] Resource usage acceptable

## Screenshots (if applicable)

<!-- Add screenshots for UI changes -->

| Before | After |
| ------ | ----- |
| ...    | ...   |

## Deployment Notes

<!-- Any special deployment considerations, configuration changes, or migration steps -->

- [ ] No database migrations required
- [ ] No configuration changes required
- [ ] No breaking API changes
- [ ] Backward compatible

### Configuration Changes

<!-- List any new or changed configuration options -->

```yaml
# Example configuration changes
```

## Checklist

<!-- Ensure all items are checked before requesting review -->

### Pre-Review

- [ ] Self-review completed
- [ ] Code is well-organized and readable
- [ ] Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
- [ ] Branch is up to date with target branch
- [ ] No merge conflicts

### Quality Gates

- [ ] All CI checks pass
- [ ] Code coverage meets requirements
- [ ] No critical security issues reported
- [ ] Documentation is complete

### Final Steps

- [ ] PR title is descriptive and follows guidelines
- [ ] Labels applied appropriately
- [ ] Reviewers assigned
- [ ] Ready for review

## Additional Notes

<!-- Any additional information for reviewers -->

## Reviewer Guidelines

<!-- For reviewers: What should be focused on during review? -->

- Focus areas:
- Known limitations:
- Questions for reviewers:
