package domain

import "time"

// TraceObservation is the provider-neutral timing subset returned by a
// read-only telemetry source. Inputs, outputs, metadata, and credentials do not
// cross this boundary.
type TraceObservation struct {
	TraceReference EvidenceReference
	SessionID      SessionID
	StartedAt      time.Time
	EndedAt        time.Time
}

// TraceObservationQuery bounds one exact-session provider read by observation
// start time. The end is exclusive to match the provider contract.
type TraceObservationQuery struct {
	SessionID     SessionID
	FromStartTime time.Time
	ToStartTime   time.Time
}
