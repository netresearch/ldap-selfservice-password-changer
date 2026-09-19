package rpchandler

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// guardInput is everything a guard may look at. It is the parsed request plus
// the client IP the server derived, never anything a guard has to re-parse.
type guardInput struct {
	params         []string
	clientIP       string
	turnstileToken string
}

// guard is a cross-cutting check that runs before a method's own logic.
//
// A guard returns stop=true when it has already written the response and the
// method must not run. The error is whatever writing that response returned,
// so a caller can treat it as an ordinary handler result.
type guard func(h *Handler, c fiber.Ctx, in guardInput) (stop bool, err error)

// methodPolicies lists, per RPC method, the guards that run before it and the
// order they run in.
//
// The order is the security property, not a detail: every local check comes
// before verifyTurnstile, so an unauthenticated flood is rejected from memory
// instead of being turned into outbound requests to Cloudflare. A method
// missing from this map reaches no guard at all, which is why Handle rejects
// an unknown method before consulting it — see
// TestUnknownMethodIsRejectedBeforeGuards.
var methodPolicies = map[string][]guard{
	"change-password": {
		guardChangePasswordIPLimit,
		guardTurnstile,
	},
	"request-password-reset": {
		guardResetServicesEnabled,
		guardResetRequestIPLimit,
		guardResetRequestIdentifierLimit,
		guardTurnstile,
	},
	"reset-password": {
		guardResetServicesEnabled,
		guardResetPasswordIPLimit,
		guardTurnstile,
	},
}

// runGuards evaluates a method's guards in order, stopping at the first one
// that answers the request.
func (h *Handler) runGuards(c fiber.Ctx, method string, in guardInput) (bool, error) {
	for _, g := range methodPolicies[method] {
		if stop, err := g(h, c, in); stop {
			return true, err
		}
	}

	return false, nil
}

// guardResetServicesEnabled rejects the password-reset methods when the
// deployment did not configure the reset services.
func guardResetServicesEnabled(h *Handler, c fiber.Ctx, _ guardInput) (bool, error) {
	if h.tokenStore != nil {
		return false, nil
	}

	return true, sendErrorResponse(c, http.StatusBadRequest, "password reset feature not enabled")
}

// guardChangePasswordIPLimit applies the per-IP limiter to password changes.
func guardChangePasswordIPLimit(h *Handler, c fiber.Ctx, in guardInput) (bool, error) {
	if h.ipLimiter == nil || h.ipLimiter.AllowRequest(in.clientIP) {
		return false, nil
	}

	slog.Warn("password_change_ip_rate_limited", "ip", in.clientIP, "username", firstParam(in.params))

	return true, sendErrorResponse(
		c,
		http.StatusTooManyRequests,
		"too many password change attempts from your IP address, please try again later",
	)
}

// guardResetPasswordIPLimit applies the per-IP limiter to a reset that redeems
// a token.
func guardResetPasswordIPLimit(h *Handler, c fiber.Ctx, in guardInput) (bool, error) {
	if h.ipLimiter == nil || h.ipLimiter.AllowRequest(in.clientIP) {
		return false, nil
	}

	return true, sendErrorResponse(
		c,
		http.StatusTooManyRequests,
		"too many password reset attempts from your IP address, please try again later",
	)
}

// guardResetRequestIPLimit applies the per-IP limiter to a reset request.
//
// Unlike the other two it answers with the same generic success a served
// request gets: this method must not let a caller distinguish outcomes, or the
// answer itself would tell an attacker which identifiers exist.
func guardResetRequestIPLimit(h *Handler, c fiber.Ctx, in guardInput) (bool, error) {
	if h.ipLimiter == nil || h.ipLimiter.AllowRequest(in.clientIP) {
		return false, nil
	}

	slog.Warn("password_reset_ip_rate_limited", "ip", in.clientIP)

	return true, sendSuccessResponse(c, []string{msgResetEmailSent})
}

// guardResetRequestIdentifierLimit applies the per-identifier limiter to the
// identifier as typed, before any outbound verification and before the
// directory lookup that would resolve it to an account.
//
// The "typed:" prefix keeps these buckets disjoint from the post-resolution
// "account:" buckets, so no typed input can address an account bucket. A
// request with the wrong parameter count passes through to the method, which
// rejects it.
func guardResetRequestIdentifierLimit(h *Handler, c fiber.Ctx, in guardInput) (bool, error) {
	if len(in.params) != 1 {
		return false, nil
	}

	if h.rateLimiter.AllowRequest("typed:" + in.params[0]) {
		return false, nil
	}

	slog.Warn("password_reset_rate_limited", "email", in.params[0])

	return true, sendSuccessResponse(c, []string{msgResetEmailSent})
}

// guardTurnstile verifies the Cloudflare Turnstile token when the feature is
// enabled. It is last in every policy, because it is the only guard that
// leaves the process.
func guardTurnstile(h *Handler, c fiber.Ctx, in guardInput) (bool, error) {
	if err := h.verifyTurnstile(in.turnstileToken, in.clientIP); err != nil {
		return true, sendErrorResponse(c, http.StatusForbidden, msgTurnstileVerificationFailed)
	}

	return false, nil
}

func firstParam(params []string) string {
	if len(params) == 0 {
		return ""
	}

	return params[0]
}
