package domain

import "time"

type Entry struct {
	TrackingID string
	AccountID  int64
	OrgID      string
	Impacts    []Impact
	CreatedAt  time.Time
}
