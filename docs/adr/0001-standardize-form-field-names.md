# ADR 0001: Standardize Form Field Names for Password Manager Compatibility

**Status**: Accepted
**Date**: 2025-10-08
**Authors**: Sebastian Mendel

## Context

The LDAP self-service password changer application contains multiple password-related forms:
- Password change form (requires current password + new password)
- Password reset request form (email input)
- Password reset form (set new password via token)

Password managers (1Password, LastPass, Bitwarden, etc.) and browser autofill features rely on specific naming conventions and autocomplete attributes to correctly detect and populate form fields. Non-standard field names reduce user experience by requiring manual field selection or preventing autofill entirely.

### Previous Implementation

The application used non-standard `name` attributes:
- `name="current"` for current password
- `name="new"` for new password
- `name="new2"` for password confirmation

While `autocomplete` attributes were correctly set, the inconsistent naming conventions created maintainability issues and reduced compatibility with some password managers that rely on field names as a detection heuristic.

## Decision

Adopt standardized HTML form field naming conventions that align with the HTML Living Standard autocomplete specification and password manager best practices:

### Standard Field Names

| Field Type | `name` Attribute | `autocomplete` Attribute |
|------------|------------------|--------------------------|
| Username/Login | `username`, `email`, or `login` | `username` or `email` |
| Current Password | `current_password` | `current-password` |
| New Password | `new_password` | `new-password` |
| Confirm Password | `confirm_password` | `new-password` |
| Email (standalone) | `email` | `email` |

### Implementation

**Templates Changed:**
- `templates/index.html`: Updated `current` → `current_password`, `new` → `new_password`, `new2` → `confirm_password`
- `templates/reset-password.html`: Updated `new` → `new_password`, `new2` → `confirm_password`
- `templates/forgot-password.html`: No changes needed (already using `email`)

**JavaScript Changed:**
- `static/js/app.ts`: Updated field definition array to reference new field names
- `static/js/reset-password.ts`: Updated field definition array to reference new field names

**Backend Impact:**
- No changes required to Go backend handlers
- Application uses JSON-RPC with positional parameters, not named form fields
- JavaScript sends data as `params: [username, oldPassword, newPassword]`

## Consequences

### Positive

1. **Improved Password Manager Compatibility**: Standard naming conventions ensure reliable detection across all major password managers (1Password, Bitwarin, LastPass, Dashlane, KeePass, etc.)

2. **Better Browser Autofill**: Modern browsers use both `autocomplete` attributes AND field names for autofill detection; standardized names improve reliability

3. **Enhanced Accessibility**: Assistive technologies benefit from consistent, predictable field naming patterns

4. **Maintainability**: Standard conventions make code more understandable for developers familiar with web standards

5. **Future-Proofing**: Alignment with HTML Living Standard ensures compatibility with future browser updates

### Neutral

1. **No Backend Changes Required**: JSON-RPC architecture with positional parameters decouples frontend field names from backend logic

2. **Client-Side Only**: Changes confined to templates and TypeScript validation logic

### Negative

1. **Migration Consideration**: If users have browser-saved credentials with old field names, they may need to re-save passwords (minimal impact due to autocomplete attributes already being correct)

## References

- [HTML Living Standard - Autofill](https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#autofill)
- [MDN Web Docs - autocomplete attribute](https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/autocomplete)
- [Password Manager Field Detection Heuristics](https://www.chromium.org/developers/design-documents/form-styles-that-chromium-understands/)
- [WCAG 2.1 - Input Purposes](https://www.w3.org/WAI/WCAG21/Understanding/identify-input-purpose.html)

## Alternatives Considered

### Alternative 1: Keep Non-Standard Names, Rely Only on Autocomplete
- **Rejected**: Password managers use field names as fallback detection mechanism when autocomplete attributes are ambiguous or missing in DOM manipulation scenarios

### Alternative 2: Use Single-Character Names (p, p1, p2)
- **Rejected**: Cryptic naming reduces maintainability and provides zero semantic value for assistive technologies

### Alternative 3: Use Framework-Specific Conventions
- **Rejected**: Web standards provide better long-term compatibility than framework-specific patterns
