# Accessibility Guide

## Overview

This application achieves **WCAG 2.2 Level AAA compliance** with adaptive features that respond to user preferences and device capabilities. This guide explains the accessibility features, compliance details, and testing procedures.

## WCAG 2.2 Compliance Summary

### Level AAA Criteria Met

| Criterion | Level | Description                 | Implementation                          |
| --------- | ----- | --------------------------- | --------------------------------------- |
| **1.4.6** | AAA   | Contrast (Enhanced)         | 7:1 contrast ratio for all text         |
| **1.4.8** | AAA   | Visual Presentation         | Enhanced text spacing, generous padding |
| **2.3.3** | AAA   | Animation from Interactions | Respects `prefers-reduced-motion`       |
| **2.5.5** | AAA   | Target Size (Enhanced)      | 44×44px minimum (comfortable mode)      |

### Level AA Criteria Met

| Criterion  | Level | Description           | Implementation                            |
| ---------- | ----- | --------------------- | ----------------------------------------- |
| **1.4.3**  | AA    | Contrast (Minimum)    | Exceeds with 7:1 ratio                    |
| **1.4.10** | AA    | Reflow                | Responsive design, no horizontal scroll   |
| **1.4.11** | AA    | Non-text Contrast     | 3:1 for UI components                     |
| **1.4.12** | AA    | Text Spacing          | User-adjustable spacing supported         |
| **2.4.7**  | AA    | Focus Visible         | Enhanced focus indicators on all elements |
| **2.5.8**  | AA    | Target Size (Minimum) | 36×36px minimum (compact mode)            |
| **4.1.2**  | AA    | Name, Role, Value     | Comprehensive ARIA attributes             |
| **4.1.3**  | AA    | Status Messages       | ARIA live regions for validation          |

## Accessibility Features

### 1. Adaptive Dark Mode

Three-state theme system that adapts to user preferences:

- **Auto Mode**: Automatically follows system `prefers-color-scheme`
- **Light Mode**: High-contrast light theme (7:1 ratio)
- **Dark Mode**: High-contrast dark theme (7:1 ratio)

**Reactive Monitoring**: Changes automatically when system preference changes without page reload.

**Color System**:

```css
/* Light mode */
--color-gray-900: oklch(15% 0.034 264.665); /* Text */
--color-gray-100: oklch(99.5% 0.003 264.542); /* Background */
/* Contrast ratio: 7:1 (AAA) */

/* Dark mode: inverted with same 7:1 ratio */
```

**Adaptive Contrast**:

- **High Contrast Mode** (`prefers-contrast: more`): Increases to `oklch(10% 0.04 264.665)` with thicker borders
- **Low Contrast Mode** (`prefers-contrast: less`): Reduces to `oklch(25% 0.02 264.665)` for light sensitivity

### 2. Adaptive Density System

Three-state density system that adapts to device capabilities:

- **Auto Mode**: Automatically detects touch devices and high contrast preferences
- **Comfortable Mode**: Spacious layout (WCAG AAA)
- **Compact Mode**: Condensed layout (WCAG AA)

#### Comfortable Mode (AAA)

- ✅ All labels visible above inputs
- ✅ 44×44px minimum touch targets (2.5.5 AAA)
- ✅ Enhanced spacing and padding
- ✅ Visible help buttons with tooltips
- ✅ Generous line-height and letter-spacing

**Automatic Activation**:

- Touch devices (`pointer: coarse`)
- High contrast preference (`prefers-contrast: more`)

#### Compact Mode (AA)

- ✅ Labels hidden visually but accessible to screen readers (`sr-only`)
- ✅ 36×36px minimum touch targets (2.5.8 AA)
- ✅ Optimized spacing for mouse/trackpad users
- ✅ Help buttons hidden (requirements in tooltips)

**Automatic Activation**:

- Mouse/trackpad devices (`pointer: fine`)
- Normal contrast preference

### 3. Keyboard Navigation

Full keyboard accessibility:

- **Tab**: Navigate forward through interactive elements
- **Shift+Tab**: Navigate backward
- **Enter/Space**: Activate buttons and toggles
- **Escape**: Close help tooltips
- **Arrow Keys**: Navigate within groups (future enhancement)

All interactive elements have visible focus indicators that exceed WCAG 2.2 requirements.

### 4. Screen Reader Support

Comprehensive ARIA implementation:

- **Landmark Regions**: `<header>`, `<main>`, `<form>` with proper roles
- **Dynamic States**: Real-time updates via ARIA live regions
- **Descriptive Labels**: All form fields have accessible labels
- **Error Messaging**: Field-specific validation with `aria-describedby`
- **Button States**: Theme and density toggles with descriptive `aria-label`

**Example ARIA Labels**:

```html
<!-- Density toggle -->
<button
  aria-label="Density: Auto density (follows system preferences).
                     Click to switch to Comfortable mode (WCAG AAA,
                     spacious layout with all labels)"
>
  <!-- Password reveal -->
  <button aria-label="Show password" aria-pressed="false">
    <!-- Form validation -->
    <input aria-invalid="false" aria-describedby="help-new errors-new" />
  </button>
</button>
```

### 5. Motion & Animation

Respects user motion preferences (WCAG 2.3.3 AAA):

```css
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

All animations disabled instantly for users with motion sensitivity.

### 6. Context-Sensitive Help

Password requirement tooltips with proper accessibility:

- **Visible in Comfortable Mode**: Help buttons (?) next to password fields
- **Hidden in Compact Mode**: Requirements shown in ARIA descriptions
- **Keyboard Accessible**: Focusable with Enter/Space to toggle
- **Screen Reader Friendly**: `aria-describedby` links to detailed requirements

### 7. Password Manager Support

Optimized for credential managers:

```html
<input type="text" name="username" autocomplete="username" />

<input type="password" name="current" autocomplete="current-password" />

<input type="password" name="new" autocomplete="new-password" />
```

## Testing Procedures

### Automated Testing

#### Contrast Ratio Verification

**Tools**:

- [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/)
- Browser DevTools Accessibility Inspector

**Expected Results**:

- All text: ≥7:1 (AAA)
- UI components: ≥3:1 (AA)
- Focus indicators: ≥3:1 (AA)

#### Keyboard Navigation

**Test**:

1. Disconnect mouse
2. Use Tab to navigate through all elements
3. Verify all interactive elements reachable
4. Verify visible focus indicators on all elements

**Expected Results**:

- All buttons, inputs, and links focusable
- Focus indicators visible with 2px outline
- Logical tab order (top to bottom, left to right)

### Screen Reader Testing

**Recommended Tools**:

- **NVDA** (Windows, free)
- **JAWS** (Windows, commercial)
- **VoiceOver** (macOS, built-in)
- **TalkBack** (Android, built-in)

#### Test Checklist

1. **Page Structure**
   - [ ] Proper headings hierarchy (H1 → H2)
   - [ ] Landmark regions announced correctly
   - [ ] Form label associations working

2. **Interactive Elements**
   - [ ] Theme toggle state announced ("Dark mode activated")
   - [ ] Density toggle state announced with detailed description
   - [ ] Password reveal button state changes ("Show password" / "Hide password")
   - [ ] Form validation errors announced in context

3. **Dynamic Content**
   - [ ] Validation errors announced when they appear
   - [ ] Help tooltips announced when opened
   - [ ] Loading states announced ("Form is busy")

### System Preference Testing

#### Dark Mode

**Test**:

1. Set density toggle to "Auto"
2. Change system dark mode preference
3. Verify application adapts without reload

**Operating Systems**:

- **Windows**: Settings → Personalization → Colors → Choose your color
- **macOS**: System Preferences → Appearance → Light/Dark
- **Linux**: Varies by desktop environment

#### High Contrast

**Test**:

1. Enable high contrast mode
2. Verify application adapts to enhanced contrast
3. Check borders become thicker

**Operating Systems**:

- **Windows**: Settings → Accessibility → Contrast themes
- **macOS**: System Preferences → Accessibility → Display → Increase contrast

#### Reduced Motion

**Test**:

1. Enable reduced motion preference
2. Interact with toggles and form elements
3. Verify no visual motion/animations

**Operating Systems**:

- **Windows**: Settings → Accessibility → Visual effects → Animation effects (OFF)
- **macOS**: System Preferences → Accessibility → Display → Reduce motion
- **Linux**: Varies by desktop environment

### Touch Device Testing

#### Comfortable Mode Auto-Activation

**Test**:

1. Set density to "Auto"
2. Open DevTools Device Toolbar (Ctrl+Shift+M)
3. Select mobile device (iPhone, Android)
4. Verify density switches to "Comfortable"

**Expected Results**:

- Density automatically switches to Comfortable mode
- All labels visible
- Touch targets ≥44×44px
- Help buttons visible

#### Compact Mode Auto-Activation

**Test**:

1. Set density to "Auto"
2. Use desktop browser with mouse
3. Verify density switches to "Compact"

**Expected Results**:

- Density automatically switches to Compact mode
- Labels hidden visually (but screen reader accessible)
- Touch targets ≥36×36px
- Help buttons hidden

### Manual Accessibility Audit

Use browser DevTools Accessibility Inspector:

**Chrome**:

1. Open DevTools (F12)
2. Go to "Lighthouse" tab
3. Check "Accessibility"
4. Generate report

**Firefox**:

1. Open DevTools (F12)
2. Go to "Accessibility" tab
3. Enable accessibility features
4. Inspect element tree

**Expected Results**:

- 100% accessibility score (or close to it)
- No contrast errors
- No missing ARIA attributes
- Proper semantic HTML structure

## Common Accessibility Patterns

### Focus Management

All interactive elements receive visible focus:

```css
button:focus {
  outline: 2px solid var(--color-gray-900);
  outline-offset: 2px;
}
```

### Error Handling

Field-specific error messages with ARIA:

```html
<input aria-invalid="true" aria-describedby="errors-username" />
<div id="errors-username" role="alert">Username must not be empty</div>
```

### Live Regions

Dynamic content changes announced:

```html
<div role="region" aria-live="polite">Password requirements appear here</div>
```

## Accessibility Resources

### WCAG 2.2 Quick Reference

- [Official WCAG 2.2 Guidelines](https://www.w3.org/WAI/WCAG22/quickref/)
- [Level AAA Filter](https://www.w3.org/WAI/WCAG22/quickref/?currentsidebar=%23col_customize&levels=aaa)

### Testing Tools

- [axe DevTools](https://www.deque.com/axe/devtools/) - Browser extension
- [WAVE](https://wave.webaim.org/) - Web accessibility evaluation tool
- [Lighthouse](https://developers.google.com/web/tools/lighthouse) - Built into Chrome DevTools

### Screen Readers

- [NVDA](https://www.nvaccess.org/) - Free, Windows
- [VoiceOver](https://www.apple.com/accessibility/voiceover/) - Free, macOS/iOS
- [TalkBack](https://support.google.com/accessibility/android/answer/6283677) - Free, Android

### Color Contrast

- [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/)
- [Contrast Ratio Calculator](https://contrast-ratio.com/)

## Support & Feedback

If you encounter accessibility issues:

1. Check this documentation for expected behavior
2. Test with latest browser version
3. Report issues on GitHub with:
   - Browser and version
   - Operating system
   - Assistive technology used (if applicable)
   - Steps to reproduce
   - Expected vs actual behavior

## Future Enhancements

Potential accessibility improvements:

- **Skip Links**: Add "Skip to main content" link
- **Focus Trap**: Modal dialogs with focus management
- **Keyboard Shortcuts**: Customizable shortcuts for power users
- **Language Support**: Multi-language support with proper `lang` attributes
- **Font Size**: User-controlled font size adjustment
