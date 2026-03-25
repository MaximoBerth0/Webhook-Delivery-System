package storage

import "errors"

var ErrEventNotFound = errors.New("event not found")

var ErrWebhookNotFound = errors.New("webhook not found")

var ErrDeliveryNotFound = errors.New("delivery not found")
