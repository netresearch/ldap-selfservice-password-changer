// Package turnstile provides Cloudflare Turnstile token verification.
package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const siteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type verifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify validates a Turnstile token with Cloudflare.
func Verify(secret, token, remoteIP string, timeout time.Duration) error {
	return VerifyWithEndpoint(siteverifyURL, secret, token, remoteIP, timeout)
}

// VerifyWithEndpoint validates a Turnstile token using the provided verification endpoint.
func VerifyWithEndpoint(endpoint, secret, token, remoteIP string, timeout time.Duration) error {
	if token == "" {
		return errors.New("missing Turnstile token")
	}

	values := url.Values{
		"secret":   {secret},
		"response": {token},
	}

	if remoteIP != "" {
		values.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		endpoint,
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		return fmt.Errorf("create Turnstile verification request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("verify Turnstile token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("turnstile verification returned HTTP %d", resp.StatusCode)
	}

	var result verifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode Turnstile verification response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf(
			"turnstile verification failed: error-codes=%v",
			result.ErrorCodes,
		)
	}

	return nil
}
