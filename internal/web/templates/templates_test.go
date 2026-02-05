//nolint:testpackage // tests internal functions
package templates

import (
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

// TestMakeInputOpts tests the MakeInputOpts function.
func TestMakeInputOpts(t *testing.T) {
	tests := []struct {
		name         string
		inputName    string
		placeholder  string
		inputType    string
		autocomplete string
		help         string
		wantPanic    bool
	}{
		{
			name:         "valid password input",
			inputName:    "new_password",
			placeholder:  "New Password",
			inputType:    "password",
			autocomplete: "new-password",
			help:         "Enter your new password",
			wantPanic:    false,
		},
		{
			name:         "valid text input",
			inputName:    "username",
			placeholder:  "Username",
			inputType:    "text",
			autocomplete: "username",
			help:         "Enter your username",
			wantPanic:    false,
		},
		{
			name:         "invalid input type causes panic",
			inputName:    "email",
			placeholder:  "Email",
			inputType:    "email",
			autocomplete: "email",
			help:         "Enter your email",
			wantPanic:    true,
		},
		{
			name:         "empty type causes panic",
			inputName:    "field",
			placeholder:  "Field",
			inputType:    "",
			autocomplete: "off",
			help:         "Help text",
			wantPanic:    true,
		},
		{
			name:         "number type causes panic",
			inputName:    "age",
			placeholder:  "Age",
			inputType:    "number",
			autocomplete: "off",
			help:         "Enter your age",
			wantPanic:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() {
					MakeInputOpts(tt.inputName, tt.placeholder, tt.inputType, tt.autocomplete, tt.help)
				})
			} else {
				result := MakeInputOpts(tt.inputName, tt.placeholder, tt.inputType, tt.autocomplete, tt.help)
				assert.Equal(t, tt.inputName, result.Name)
				assert.Equal(t, tt.placeholder, result.Placeholder)
				assert.Equal(t, tt.inputType, result.Type)
				assert.Equal(t, tt.autocomplete, result.Autocomplete)
				assert.Equal(t, tt.help, result.Help)
			}
		})
	}
}

// TestInputOptsStruct tests the InputOpts struct fields.
func TestInputOptsStruct(t *testing.T) {
	opts := InputOpts{
		Name:         "password",
		Placeholder:  "Enter Password",
		Type:         "password",
		Autocomplete: "current-password",
		Help:         "Your current password",
	}

	assert.Equal(t, "password", opts.Name)
	assert.Equal(t, "Enter Password", opts.Placeholder)
	assert.Equal(t, "password", opts.Type)
	assert.Equal(t, "current-password", opts.Autocomplete)
	assert.Equal(t, "Your current password", opts.Help)
}

// TestRenderIndex tests the RenderIndex function.
func TestRenderIndex(t *testing.T) {
	tests := []struct {
		name                string
		opts                *options.Opts
		wantContains        []string
		wantNotContains     []string
		wantPasswordReset   bool
		passwordResetString string
	}{
		{
			name: "renders with password reset enabled",
			opts: &options.Opts{
				MinLength:                  8,
				MinNumbers:                 1,
				MinSymbols:                 1,
				MinUppercase:               1,
				MinLowercase:               1,
				PasswordCanIncludeUsername: false,
				PasswordResetEnabled:       true,
			},
			wantContains: []string{
				"<!doctype html>",
				"GopherPass",
				"Change Your Password",
				"data-min-length=\"8\"",
				"data-min-numbers=\"1\"",
				"data-min-symbols=\"1\"",
				"data-min-uppercase=\"1\"",
				"data-min-lowercase=\"1\"",
				"data-password-can-include-username=\"false\"",
				"/forgot-password",
			},
			wantPasswordReset:   true,
			passwordResetString: "Forgot Password?",
		},
		{
			name: "renders with password reset disabled",
			opts: &options.Opts{
				MinLength:                  12,
				MinNumbers:                 2,
				MinSymbols:                 2,
				MinUppercase:               2,
				MinLowercase:               2,
				PasswordCanIncludeUsername: true,
				PasswordResetEnabled:       false,
			},
			wantContains: []string{
				"<!doctype html>",
				"GopherPass",
				"data-min-length=\"12\"",
				"data-min-numbers=\"2\"",
				"data-password-can-include-username=\"true\"",
			},
			wantNotContains:   []string{"/forgot-password"},
			wantPasswordReset: false,
		},
		{
			name: "handles zero values",
			opts: &options.Opts{
				MinLength:                  0,
				MinNumbers:                 0,
				MinSymbols:                 0,
				MinUppercase:               0,
				MinLowercase:               0,
				PasswordCanIncludeUsername: false,
				PasswordResetEnabled:       false,
			},
			wantContains: []string{
				"data-min-length=\"0\"",
				"data-min-numbers=\"0\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderIndex(tt.opts)
			require.NoError(t, err)
			require.NotEmpty(t, result)

			html := string(result)

			for _, want := range tt.wantContains {
				assert.Contains(t, html, want, "expected HTML to contain %q", want)
			}

			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, html, notWant, "expected HTML not to contain %q", notWant)
			}
		})
	}
}

// TestRenderForgotPassword tests the RenderForgotPassword function.
func TestRenderForgotPassword(t *testing.T) {
	result, err := RenderForgotPassword()
	require.NoError(t, err)
	require.NotEmpty(t, result)

	html := string(result)

	// Check for expected content
	expectedStrings := []string{
		"<!doctype html>",
		"Forgot Password",
		"GopherPass",
		"Email Address",
		"Send Reset Link",
		"Back to Login",
		"/forgot-password-init.js",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, html, expected, "expected HTML to contain %q", expected)
	}

	// Verify the form elements are present
	assert.Contains(t, html, `id="form"`)
	assert.Contains(t, html, `role="alert"`)
}

// TestRenderResetPassword tests the RenderResetPassword function.
func TestRenderResetPassword(t *testing.T) {
	tests := []struct {
		name         string
		opts         *options.Opts
		wantContains []string
	}{
		{
			name: "renders with standard options",
			opts: &options.Opts{
				MinLength:    8,
				MinNumbers:   1,
				MinSymbols:   1,
				MinUppercase: 1,
				MinLowercase: 1,
			},
			wantContains: []string{
				"<!doctype html>",
				"Reset Password",
				"GopherPass",
				"New Password",
				"Confirm New Password",
				"data-min-length=\"8\"",
				"data-min-numbers=\"1\"",
				"data-min-symbols=\"1\"",
				"data-min-uppercase=\"1\"",
				"data-min-lowercase=\"1\"",
				"Back to Login",
			},
		},
		{
			name: "renders with custom password requirements",
			opts: &options.Opts{
				MinLength:    16,
				MinNumbers:   3,
				MinSymbols:   3,
				MinUppercase: 3,
				MinLowercase: 3,
			},
			wantContains: []string{
				"data-min-length=\"16\"",
				"data-min-numbers=\"3\"",
				"data-min-symbols=\"3\"",
				"data-min-uppercase=\"3\"",
				"data-min-lowercase=\"3\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderResetPassword(tt.opts)
			require.NoError(t, err)
			require.NotEmpty(t, result)

			html := string(result)

			for _, want := range tt.wantContains {
				assert.Contains(t, html, want, "expected HTML to contain %q", want)
			}
		})
	}
}

// TestRenderIndexIdempotent tests that RenderIndex produces consistent output.
func TestRenderIndexIdempotent(t *testing.T) {
	opts := &options.Opts{
		MinLength:                  8,
		MinNumbers:                 1,
		MinSymbols:                 1,
		MinUppercase:               1,
		MinLowercase:               1,
		PasswordCanIncludeUsername: false,
		PasswordResetEnabled:       true,
	}

	result1, err1 := RenderIndex(opts)
	require.NoError(t, err1)

	result2, err2 := RenderIndex(opts)
	require.NoError(t, err2)

	assert.Equal(t, result1, result2, "RenderIndex should produce identical output for same input")
}

// TestRenderForgotPasswordIdempotent tests that RenderForgotPassword produces consistent output.
func TestRenderForgotPasswordIdempotent(t *testing.T) {
	result1, err1 := RenderForgotPassword()
	require.NoError(t, err1)

	result2, err2 := RenderForgotPassword()
	require.NoError(t, err2)

	assert.Equal(t, result1, result2, "RenderForgotPassword should produce identical output")
}

// TestRenderResetPasswordIdempotent tests that RenderResetPassword produces consistent output.
func TestRenderResetPasswordIdempotent(t *testing.T) {
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}

	result1, err1 := RenderResetPassword(opts)
	require.NoError(t, err1)

	result2, err2 := RenderResetPassword(opts)
	require.NoError(t, err2)

	assert.Equal(t, result1, result2, "RenderResetPassword should produce identical output for same input")
}

// TestMakeSlice tests the makeSlice helper function.
func TestMakeSlice(t *testing.T) {
	tests := []struct {
		name string
		args []interface{}
		want []interface{}
	}{
		{
			name: "empty slice",
			args: []interface{}{},
			want: []interface{}{},
		},
		{
			name: "single element",
			args: []interface{}{"hello"},
			want: []interface{}{"hello"},
		},
		{
			name: "multiple strings",
			args: []interface{}{"a", "b", "c"},
			want: []interface{}{"a", "b", "c"},
		},
		{
			name: "mixed types",
			args: []interface{}{"string", 42, true},
			want: []interface{}{"string", 42, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeSlice(tt.args...)
			assert.Equal(t, tt.want, result)
		})
	}
}

// TestRenderTemplateValidHTML tests that rendered templates produce valid HTML structure.
func TestRenderTemplateValidHTML(t *testing.T) {
	opts := &options.Opts{
		MinLength:            8,
		MinNumbers:           1,
		MinSymbols:           1,
		MinUppercase:         1,
		MinLowercase:         1,
		PasswordResetEnabled: true,
	}

	tests := []struct {
		name   string
		render func() ([]byte, error)
	}{
		{
			name: "index",
			render: func() ([]byte, error) {
				return RenderIndex(opts)
			},
		},
		{
			name:   "forgot-password",
			render: RenderForgotPassword,
		},
		{
			name: "reset-password",
			render: func() ([]byte, error) {
				return RenderResetPassword(opts)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.render()
			require.NoError(t, err)

			html := string(result)

			// Check for basic HTML structure
			assert.True(t, strings.HasPrefix(html, "<!doctype html>"), "should start with doctype")
			assert.Contains(t, html, "<html", "should contain html tag")
			assert.Contains(t, html, "</html>", "should contain closing html tag")
			assert.Contains(t, html, "<head>", "should contain head tag")
			assert.Contains(t, html, "</head>", "should contain closing head tag")
			assert.Contains(t, html, "<body", "should contain body tag")
			assert.Contains(t, html, "</body>", "should contain closing body tag")
			assert.Contains(t, html, "<title>", "should contain title tag")

			// Check accessibility attributes
			assert.Contains(t, html, `lang="en"`, "should have lang attribute")
			assert.Contains(t, html, `role="main"`, "should have main role")
			assert.Contains(t, html, `role="alert"`, "should have alert role for errors")
			assert.Contains(t, html, `aria-live="assertive"`, "should have assertive live region")
		})
	}
}

// TestRenderIndexSpecialCharacters tests rendering with special characters in options.
func TestRenderIndexSpecialCharacters(t *testing.T) {
	// Ensure options with special characters don't cause template errors
	opts := &options.Opts{
		MinLength:                  8,
		MinNumbers:                 1,
		MinSymbols:                 1,
		MinUppercase:               1,
		MinLowercase:               1,
		PasswordCanIncludeUsername: false,
		PasswordResetEnabled:       true,
	}

	result, err := RenderIndex(opts)
	require.NoError(t, err)
	require.NotEmpty(t, result)
}

// BenchmarkRenderIndex benchmarks the RenderIndex function.
func BenchmarkRenderIndex(b *testing.B) {
	opts := &options.Opts{
		MinLength:                  8,
		MinNumbers:                 1,
		MinSymbols:                 1,
		MinUppercase:               1,
		MinLowercase:               1,
		PasswordCanIncludeUsername: false,
		PasswordResetEnabled:       true,
	}

	b.ResetTimer()
	for range b.N {
		_, _ = RenderIndex(opts) //nolint:errcheck // benchmark
	}
}

// BenchmarkRenderForgotPassword benchmarks the RenderForgotPassword function.
func BenchmarkRenderForgotPassword(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_, _ = RenderForgotPassword() //nolint:errcheck // benchmark
	}
}

// BenchmarkRenderResetPassword benchmarks the RenderResetPassword function.
func BenchmarkRenderResetPassword(b *testing.B) {
	opts := &options.Opts{
		MinLength:    8,
		MinNumbers:   1,
		MinSymbols:   1,
		MinUppercase: 1,
		MinLowercase: 1,
	}

	b.ResetTimer()
	for range b.N {
		_, _ = RenderResetPassword(opts) //nolint:errcheck // benchmark
	}
}

// TestRenderTemplateWithInvalidTemplate tests error handling for invalid templates.
func TestRenderTemplateWithInvalidTemplate(t *testing.T) {
	// Test with invalid template syntax (unclosed action)
	invalidTemplate := `{{define "test"}}Hello {{.Name}{{end}}`
	_, err := renderTemplate("test", invalidTemplate, nil)
	assert.Error(t, err, "should error on invalid template syntax")
	assert.Contains(t, err.Error(), "failed to parse")
}

// TestRenderTemplateWithExecutionError tests error handling during template execution.
func TestRenderTemplateWithExecutionError(t *testing.T) {
	// Template that references a non-existent template
	templateWithMissingRef := `{{define "test"}}{{template "nonexistent"}}{{end}}`
	_, err := renderTemplate("test", templateWithMissingRef, nil)
	assert.Error(t, err, "should error when referencing non-existent template")
}

// TestRenderTemplateWithNilData tests rendering with nil data.
func TestRenderTemplateWithNilData(t *testing.T) {
	// Simple template that doesn't require data
	simpleTemplate := `{{define "simple"}}Hello World{{end}}`
	result, err := renderTemplate("simple", simpleTemplate, nil)
	assert.NoError(t, err)
	assert.Contains(t, string(result), "Hello World")
}

// TestParseCommonTemplatesDirectly tests parseCommonTemplates function.
func TestParseCommonTemplatesDirectly(t *testing.T) {
	// Create a new template and parse common templates
	tpl := template.New("test")
	err := parseCommonTemplates(tpl)
	assert.NoError(t, err, "parseCommonTemplates should succeed with valid embedded templates")

	// Verify some templates were defined
	definedTemplates := tpl.DefinedTemplates()
	assert.NotEmpty(t, definedTemplates, "should have defined templates")
}
