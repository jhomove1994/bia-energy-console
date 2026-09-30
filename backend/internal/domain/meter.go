// Package domain holds the platform entities and enums. It has no dependencies.
package domain

import "time"

// Meter is identified only by its id; no name or location is stored or invented.
type Meter struct {
	MeterID   string
	CreatedAt time.Time
}
