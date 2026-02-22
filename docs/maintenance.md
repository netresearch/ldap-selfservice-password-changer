# Documentation Maintenance Guide

How to keep documentation accurate and up-to-date as the codebase evolves.

## When to Update Documentation

### Immediate Updates Required

#### Code Changes

- ✅ **New Features**: Document in [Component Reference](component-reference.md) + [API Reference](api-reference.md) if API changes
- ✅ **Configuration Changes**: Update [Development Guide: Configuration Reference](development-guide.md#configuration-reference)
- ✅ **Architecture Decisions**: Add to [Architecture Patterns](architecture-patterns.md) with rationale
- ✅ **Breaking Changes**: Update all affected docs + add migration guide
- ✅ **Security Changes**: Update [API Reference: Security Considerations](api-reference.md#security-considerations)

#### File Structure Changes

- ✅ **New Packages**: Add to [Component Reference](component-reference.md)
- ✅ **Moved Files**: Update all file path references
- ✅ **Renamed Functions**: Update [Component Reference](component-reference.md) function signatures
- ✅ **Deleted Components**: Remove from all documentation

#### Dependency Changes

- ✅ **Go Modules**: Update [Project Context](project-context-2025-10-04.md#dependencies-summary)
- ✅ **npm Packages**: Update [Project Context](project-context-2025-10-04.md#dependencies-summary)
- ✅ **Version Bumps**: Update tech stack references

### Periodic Updates Recommended

#### Quarterly Reviews

- 📅 **Q1, Q2, Q3, Q4**: Review all docs for accuracy
- 📅 **After Major Releases**: Regenerate [Project Context](project-context-2025-10-04.md)
- 📅 **Before Onboarding**: Verify [Onboarding Checklist](onboarding-checklist.md) still valid
- 📅 **Test Coverage Changes**: Update [Testing Guide](testing-guide.md)

---

## Documentation Update Workflows

### Workflow 1: Small Code Change (Function Added)

**Scenario**: Added `MinSpecialCharsInString()` validator

**Steps**:

1. Update [Component Reference: internal/validators](component-reference.md#go-package-internalvalidators)
   - Add function signature
   - Add purpose and algorithm
   - Add usage example

2. If exposed via API, update [API Reference: Validation Rules](api-reference.md#validation-rules)
   - Add new rule to table
   - Document error message

3. Update [Testing Guide](testing-guide.md) if test added
   - Add to existing test coverage

**Time Estimate**: 10-15 minutes

---

### Workflow 2: New Configuration Option

**Scenario**: Added `LDAP_TIMEOUT` environment variable

**Steps**:

1. Update [Development Guide: Configuration Reference](development-guide.md#configuration-reference)
   - Add to environment variables table
   - Add to command-line flags section
   - Add usage example

2. Update [Project Context: Configuration](project-context-2025-10-04.md#configuration)
   - Add to .env section

3. Update [Component Reference: internal/options](component-reference.md#go-package-internaloptions)
   - Add to `Opts` struct definition
   - Document parsing logic

**Time Estimate**: 15-20 minutes

---

### Workflow 3: Architecture Change

**Scenario**: Changed from JSON-RPC to RESTful endpoints

**Steps**:

1. Update [Architecture Patterns: Core Architecture](architecture-patterns.md#core-architecture)
   - Rewrite JSON-RPC section
   - Document new REST design
   - Update decision rationale

2. Update [API Reference](api-reference.md)
   - Complete rewrite of endpoint specification
   - Update request/response formats
   - Update examples

3. Update [Component Reference: internal/rpchandler](component-reference.md#go-package-internalrpc)
   - Update handler documentation
   - Update function signatures

4. Update [Development Guide](development-guide.md)
   - Update API testing examples
   - Update curl commands

**Time Estimate**: 2-3 hours

---

### Workflow 4: Major Refactoring

**Scenario**: Rewrote entire validation system

**Steps**:

1. **Option A**: Regenerate all documentation

   ```bash
   /sc:index --comprehensive
   ```

2. **Option B**: Manual updates
   - Update [Component Reference](component-reference.md) - All affected packages
   - Update [API Reference](api-reference.md) - Validation section
   - Update [Architecture Patterns](architecture-patterns.md) - Validation patterns
   - Update [Testing Guide](testing-guide.md) - Test examples
   - Update code examples in all docs

**Time Estimate**: 4-6 hours (manual) or 30 minutes (regeneration + review)

---

## Automated Documentation Regeneration

### When to Regenerate

Use `/sc:index --comprehensive` for:

- ✅ Major refactoring (>20% of codebase changed)
- ✅ New package added
- ✅ Significant architecture changes
- ✅ After version releases (to capture state)
- ✅ Documentation becomes stale (>6 months old)

### Regeneration Process

#### Step 1: Backup Current Docs

```bash
cp -r claudedocs claudedocs.backup-$(date +%Y%m%d)
```

#### Step 2: Regenerate

```bash
/sc:index --ultrathink --seq --loop --validate --concurrency 10 --comprehensive
```

#### Step 3: Review Changes

```bash
diff -r claudedocs.backup-* claudedocs/
```

#### Step 4: Merge Custom Content

- Manual sections (custom guides, team-specific notes)
- Updated examples
- Onboarding modifications

#### Step 5: Validate

- Check all cross-references
- Verify line numbers
- Test code examples
- Review for accuracy

#### Step 6: Commit

```bash
git add claudedocs/
git commit -m "docs: regenerate comprehensive documentation"
```

---

## Documentation Quality Checklist

### Before Committing Documentation Changes

- [ ] **Accuracy**: All code references verified
- [ ] **Line Numbers**: Match current codebase
- [ ] **Cross-References**: All links valid
- [ ] **Code Examples**: Tested and working
- [ ] **Formatting**: Consistent markdown
- [ ] **Spelling**: No typos
- [ ] **Completeness**: All sections updated
- [ ] **Navigation**: Index updated if needed

### Quality Standards

#### Code References

✅ **Good**: `internal/rpchandler/handler.go:33-46 - wrapRPC function`
❌ **Bad**: `handler.go - the wrapper function`

#### Code Examples

✅ **Good**: Complete, runnable examples with context
❌ **Bad**: Partial snippets without explanation

#### Cross-References

✅ **Good**: `[Architecture Patterns: Dual Validation](architecture-patterns.md#dual-validation-frontend--backend)`
❌ **Bad**: "See architecture docs"

---

## File-Specific Maintenance

### README.md

**Update Frequency**: With every new doc file
**Triggers**: New documentation created, major section added
**Sections to Update**:

- Documentation Structure (file list)
- Document Index (add new file)
- Quick Reference (if tech stack changes)

### project-context-\*.md

**Update Frequency**: Monthly or with major changes
**Triggers**: Dependency updates, architecture changes, git status changes
**Regeneration**: Create new file with timestamp
**Keep**: Historical context files for version comparison

### api-reference.md

**Update Frequency**: With every API change
**Triggers**: New RPC methods, validation rules, endpoints
**Critical Sections**:

- Available Methods
- Validation Rules table
- Error messages
- Request/response formats

### architecture-patterns.md

**Update Frequency**: With design decisions
**Triggers**: Pattern changes, new patterns adopted, trade-off analysis
**Add to**: Design Decisions Summary table

### development-guide.md

**Update Frequency**: With workflow or setup changes
**Triggers**: New tools, configuration options, common tasks
**Critical Sections**:

- Configuration Reference (env vars table)
- Common Development Tasks
- Troubleshooting

### testing-guide.md

**Update Frequency**: With test changes
**Triggers**: New test types, coverage changes, tool updates
**Critical Sections**:

- Current Test Coverage
- Test organization structure

### component-reference.md

**Update Frequency**: With code changes
**Triggers**: New functions, packages, modules, type definitions
**Critical Sections**:

- Function signatures
- Type definitions
- Usage examples

### onboarding-checklist.md

**Update Frequency**: Quarterly review
**Triggers**: Process improvements, new team feedback, prerequisite changes
**Validation**: New hire feedback

---

## Common Maintenance Tasks

### Task 1: Update Line Number References

**When**: After refactoring that changes line numbers

**Process**:

1. Identify affected files (git diff)
2. Search docs for file references:
   ```bash
   grep -r "filename.go:" claudedocs/
   ```
3. Open file and verify line numbers
4. Update all references

**Tip**: Use absolute line numbers + context description:

```markdown
internal/rpchandler/handler.go:33-46 (wrapRPC function)
```

---

### Task 2: Add New Function Documentation

**When**: New function added to codebase

**Template**:

````markdown
#### `FunctionName()`

**Signature**: `func FunctionName(param1 type1, param2 type2) returnType`

**Purpose**: Brief description of what it does

**Parameters**:

- `param1`: Description
- `param2`: Description

**Returns**: Description of return value(s)

**Example**:

```go
result := FunctionName(arg1, arg2)
```
````

**Usage in**: Where it's called

````

---

### Task 3: Document Breaking Change

**When**: API or interface changes incompatibly

**Required Documentation**:
1. **CHANGELOG.md** (if exists): Add breaking change entry
2. **API Reference**: Update affected sections with migration notes
3. **Development Guide**: Add upgrade instructions
4. **Architecture Patterns**: Document decision rationale

**Template**:
```markdown
### Breaking Change: [Description]

**Changed in**: Version X.Y.Z

**Before**:
```go
oldFunction(arg1)
````

**After**:

```go
newFunction(arg1, arg2)
```

**Migration**: How to update existing code

**Rationale**: Why the change was necessary

````

---

## Documentation Review Process

### Self-Review Checklist
Before committing documentation changes:

1. **Accuracy**
   - [ ] Tested all code examples
   - [ ] Verified all line numbers
   - [ ] Checked function signatures

2. **Completeness**
   - [ ] All new features documented
   - [ ] Cross-references added
   - [ ] Examples provided

3. **Consistency**
   - [ ] Formatting matches existing docs
   - [ ] Terminology consistent
   - [ ] Structure follows patterns

4. **Usability**
   - [ ] Navigation clear
   - [ ] Target audience appropriate
   - [ ] Examples helpful

### Peer Review (Recommended)
- Have another developer review documentation
- Verify examples work for them
- Check for clarity and completeness

---

## Version Control for Documentation

### Commit Message Guidelines

**Format**: `docs: <description>`

**Examples**:
```bash
docs: add validation rules for new password strength feature
docs: update line numbers after refactoring
docs: regenerate comprehensive documentation for v2.0
docs(api): document new rate limiting endpoint
docs(fix): correct typo in development guide
````

### Branch Strategy

**Small Changes**: Commit directly to main

```bash
git add claudedocs/api-reference.md
git commit -m "docs: update validation rules table"
git push
```

**Large Changes**: Use feature branch

```bash
git checkout -b docs/update-architecture
# Make changes
git commit -m "docs: major architecture update for microservices"
git push -u origin docs/update-architecture
# Create PR for review
```

---

## Tools and Automation

### Documentation Validation Scripts

#### Check for Broken Links

```bash
# Check all markdown links
find claudedocs -name "*.md" -exec markdown-link-check {} \;
```

#### Verify Line Number References

```bash
# Extract file:line references and verify they exist
grep -roh "[a-z/_]*\.go:[0-9]*" claudedocs/ | while read ref; do
  file=$(echo $ref | cut -d: -f1)
  line=$(echo $ref | cut -d: -f2)
  if [ ! -f "$file" ]; then
    echo "Missing file: $file"
  fi
done
```

#### Count Documentation Coverage

```bash
# Count documented vs undocumented functions
go doc -all ./... | grep "^func" | wc -l  # Total functions
grep -c "^#### \`.*\(\)`" claudedocs/component-reference.md  # Documented
```

### Pre-Commit Hook (Optional)

Create `.git/hooks/pre-commit`:

```bash
#!/bin/sh
# Validate documentation before commit

# Check for TODO markers in docs
if git diff --cached --name-only | grep "^claudedocs/.*\.md$" | xargs grep -q "TODO\|FIXME\|XXX"; then
  echo "Error: Documentation contains TODO markers"
  exit 1
fi

# Check for broken internal links (basic check)
if git diff --cached --name-only | grep "^claudedocs/.*\.md$" | xargs grep -oh "\[.*\](.*\.md.*)" | grep -v "^http" | while read link; do
  file=$(echo $link | sed -n 's/.*(\(.*\))/\1/p' | cut -d# -f1)
  if [ -n "$file" ] && [ ! -f "claudedocs/$file" ]; then
    echo "Error: Broken link to $file"
    exit 1
  fi
done; then
  exit 0
fi
```

---

## Maintenance Schedule

### Weekly

- [ ] Review recent commits for documentation gaps
- [ ] Update line numbers if code refactored
- [ ] Check GitHub issues for documentation requests

### Monthly

- [ ] Regenerate [Project Context](project-context-2025-10-04.md) with latest git status
- [ ] Review [Onboarding Checklist](onboarding-checklist.md) for accuracy
- [ ] Update dependency versions

### Quarterly

- [ ] Full documentation review (all files)
- [ ] Test all code examples
- [ ] Gather new hire feedback
- [ ] Update [Testing Guide](testing-guide.md) coverage statistics

### Annually

- [ ] Full regeneration with `/sc:index --comprehensive`
- [ ] Archive old context files
- [ ] Review documentation structure
- [ ] Update documentation standards

---

## Ownership and Responsibility

### Documentation Owners

- **API Reference**: Backend developers
- **Component Reference**: Package maintainers
- **Development Guide**: DevOps + senior developers
- **Testing Guide**: QA engineers
- **Architecture Patterns**: Tech lead
- **Onboarding Checklist**: Team lead

### Team Responsibility

Every developer is responsible for:

- Updating docs for their code changes
- Reviewing documentation in PRs
- Reporting documentation gaps
- Suggesting improvements

---

## Feedback and Improvement

### Documentation Feedback Channels

- GitHub Issues with `documentation` label
- Team retrospectives
- New hire exit interviews
- Code review comments

### Continuous Improvement

- Track time spent finding information
- Measure onboarding time
- Survey developer satisfaction
- Monitor documentation usage

---

## Emergency Documentation Updates

### Critical Bug Documentation

When a critical bug is discovered:

1. **Immediate**: Add warning to relevant documentation
2. **Within 24h**: Document workaround
3. **After fix**: Update with proper solution
4. **Post-mortem**: Add to architecture patterns if architectural

**Example Warning**:

```markdown
⚠️ **Known Issue**: Password validation fails for non-ASCII characters.
**Workaround**: Restrict passwords to ASCII-only until v2.1.0.
**Tracked**: #123
```

---

## Related Resources

- [README.md](README.md) - Documentation navigation
- [Onboarding Checklist](onboarding-checklist.md) - New developer guide
- Project README - Contributing guidelines
- Git commit conventions

---

_Generated by /sc:document for documentation maintenance - 2025-10-04_
_Last Updated: 2025-10-04_
