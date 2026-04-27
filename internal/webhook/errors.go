package webhook

import "errors"

var (
	ErrWebhookNotFound    = errors.New("webhook: webhook not found")
	ErrURLrequired        = errors.New("webhook: target URL is required")
	ErrInvalidTargetURL   = errors.New("webhook: invalid target URL")
	ErrSecretRequired     = errors.New("webhook: secret is required")
	ErrInvalidMaxAttempts = errors.New("max attempts must be >= 1")
	ErrNumberOfEvents     = errors.New("at least one event required")
	ErrInvalidEventType   = errors.New("webhook: invalid event type")
	ErrGetWebhook         = errors.New("webhook: webhook not found")
	ErrListWebhooks       = errors.New("webhook: error listing webhooks")
	ErrUpdateWebhook      = errors.New("webhook: error updating webhook")
	ErrDeleteWebhook      = errors.New("webhook: error deleting webhook")
)
