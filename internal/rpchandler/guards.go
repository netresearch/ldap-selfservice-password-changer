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

// guardFunc is a cross-cutting check that runs before a method's own logic.
//
// It returns stop=true when it has already written the response and the method
// must not run. The error is whatever writing that response returned, so a
// caller can treat it as an ordinary handler result. With stop=false the error
// is meaningless and ignored.
type guardFunc func(h *Handler, c fiber.Ctx, in guardInput) (stop bool, err error)

// guardStep is a guard under a name. The name is what the ordering test
// asserts on: comparing function pointers would reject a guard wrapped in a
// closure and, worse, would report two closures made by one factory as the
// same guard. The cost is that a name cannot vouch for the function beside it
// — a step whose name and run disagree passes the ordering test and is caught
// only by the behavioral ones.
type guardStep struct {
	name string
	run  guardFunc
}

// Guard names. turnstileGuard is the one the ordering test asserts on; the
// others are named so a failing policy reads as a sequence rather than as a
// list of addresses.
const (
	paramCountGuard       = "param-count"
	resetServicesGuard    = "reset-services"
	identifierLengthGuard = "identifier-length"
	ipLimitGuard          = "ip-limit"
	identifierLimitGuard  = "identifier-limit"
	turnstileGuard        = "turnstile"
)

// maxIdentifierLength is the RFC 5321 maximum for an email address. Anything
// longer cannot be one, and refusing it must not cost a rate-limiter slot.
const maxIdentifierLength = 254

// methodPolicies lists, per RPC method, the guards that run before it and the
// order they run in.
//
// The order is the security property, not a detail. A malformed or over-long
// request is refused first, so that rejecting it costs neither a limiter slot
// nor a verification; the in-memory limiters come next; and Turnstile is always
// last, so an unauthenticated flood is rejected from memory instead of being
// turned into outbound requests to Cloudflare.
//
// Every dispatchable method has an entry here, which
// TestEveryDispatchableMethodHasAPolicy pins: a method added to methodHandlers
// without a policy would otherwise run with no guards at all.
var methodPolicies = map[string][]guardStep{
	"change-password": {
		{name: paramCountGuard, run: guardChangePasswordParams},
		{name: ipLimitGuard, run: guardChangePasswordIPLimit},
		{name: turnstileGuard, run: guardTurnstile},
	},
	"request-password-reset": {
		{name: resetServicesGuard, run: guardResetServicesEnabled},
		{name: paramCountGuard, run: guardResetRequestParams},
		{name: identifierLengthGuard, run: guardResetRequestIdentifierLength},
		{name: ipLimitGuard, run: guardResetRequestIPLimit},
		{name: identifierLimitGuard, run: guardResetRequestIdentifierLimit},
		{name: turnstileGuard, run: guardTurnstile},
	},
	"reset-password": {
		// The parameter count is checked by resetPassword itself, after these
		// guards, exactly as it was before the chain existed.
		{name: resetServicesGuard, run: guardResetServicesEnabled},
		{name: ipLimitGuard, run: guardResetPasswordIPLimit},
		{name: turnstileGuard, run: guardTurnstile},
	},
}

// runGuards evaluates a method's guards in order, stopping at the first one
// that answers the request.
func (h *Handler) runGuards(c fiber.Ctx, method string, in guardInput) (bool, error) {
	for _, step := range methodPolicies[method] {
		if stop, err := step.run(h, c, in); stop {
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

// guardChangePasswordParams rejects a change-password call with the wrong
// parameter count. The method checks the count again; this guard exists so the
// rejection happens before the limiter and the verification, as it did when
// the check was the first statement of the method.
func guardChangePasswordParams(_ *Handler, c fiber.Ctx, in guardInput) (bool, error) {
	if len(in.params) == 3 {
		return false, nil
	}

	return true, sendErrorResponse(c, http.StatusInternalServerError, ErrInvalidArgumentCount.Error())
}

// guardResetRequestParams does the same for a reset request, and additionally
// lets every guard after it read params[0] without a length check of its own.
func guardResetRequestParams(_ *Handler, c fiber.Ctx, in guardInput) (bool, error) {
	if len(in.params) == 1 {
		return false, nil
	}

	return true, sendErrorResponse(c, http.StatusInternalServerError, ErrInvalidArgumentCount.Error())
}

// guardResetRequestIdentifierLength refuses an identifier that cannot be an
// email address. It answers like a served request, for the same reason the
// limiters below do.
func guardResetRequestIdentifierLength(_ *Handler, c fiber.Ctx, in guardInput) (bool, error) {
	if len(in.params) != 1 || len(in.params[0]) <= maxIdentifierLength {
		return false, nil
	}

	slog.Warn("password_reset_email_too_long", "length", len(in.params[0]))

	return true, sendSuccessResponse(c, []string{msgResetEmailSent})
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
// "account:" buckets, so no typed input can address an account bucket.
//
// h.rateLimiter is not checked for nil: NewWithServices sets it together with
// tokenStore, and guardResetServicesEnabled runs first in this policy, so a
// handler without one never reaches here. Moving that guard later would break
// this assumption as well as the feature check itself.
func guardResetRequestIdentifierLimit(h *Handler, c fiber.Ctx, in guardInput) (bool, error) {
	if len(in.params) != 1 || h.rateLimiter.AllowRequest("typed:"+in.params[0]) {
		return false, nil
	}

	slog.Warn("password_reset_rate_limited", "email", in.params[0])

	return true, sendSuccessResponse(c, []string{msgResetEmailSent})
}

// firstParam is the bounds-safe read the guards that log or key on the first
// parameter use. A policy always places its param-count guard first, so an
// empty slice cannot reach them today; this keeps a reordering a wrong answer
// rather than a panic, since main.go installs no recover middleware.
func firstParam(params []string) string {
	if len(params) == 0 {
		return ""
	}

	return params[0]
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
