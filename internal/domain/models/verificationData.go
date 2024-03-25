package models

import (
	"time"
)

type VerificationData struct {
	Email     string
	Code      string
	ExpiresAt time.Time
}
