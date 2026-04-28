// Package actor defines a server-side carrier for trusted actor identity.
//
// This package is the minimal foundation for the ActorContext primitive
// described in D-0054 (actor identity is server-resolved; payload actor
// fields are prototype compatibility / audit labels only).
//
// It is not wired into routes, services, or persistence yet. Existing
// payload-supplied actor fields (`request_author`, `intent_owner`,
// `submitted_by`, `applied_by`, `updated_by`, `marked_by`, `approved_by`)
// remain in force as prototype compatibility / audit labels until later
// bounded slices migrate the relevant transitions to ActorContext.
//
// This package does not implement authentication or authorization, does
// not read request headers, does not register middleware, and does not
// resolve actor identity from any transport.
package actor

import (
	"context"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

// Source identifies how an ActorContext was resolved.
//
// Source is informational only. It does not by itself authorize a
// transition; authorization remains a future bounded slice.
type Source string

// Source constants enumerate how an ActorContext may be obtained.
//
// SourcePayloadCompat marks the legacy compatibility path described in
// D-0054, where actor identity comes from a request payload field.
// Other sources represent future trusted-resolution paths and are not
// implemented in this slice.
const (
	SourceUnknown       Source = "unknown"
	SourceDevHeader     Source = "dev_header"
	SourcePayloadCompat Source = "payload_compat"
	SourceService       Source = "service"
	SourceSystem        Source = "system"
)

// ActorContext carries trusted actor identity for a server-owned
// canonical state transition. It wraps the existing spine.ActorRef and
// records how the identity was resolved.
//
// ActorContext is not yet consumed by any handler or service; later
// bounded slices will migrate highest-risk transitions (approve,
// mark ready_for_approval, apply clarification, update draft, intake
// author handling) to read identity from ActorContext rather than
// from request payload fields.
type ActorContext struct {
	Actor  spine.ActorRef
	Source Source
}

type contextKey struct{}

// WithActor returns a new context that carries the given ActorContext.
//
// The carrier is intended for server-internal use; it does not
// authenticate or authorize the actor.
func WithActor(ctx context.Context, actor ActorContext) context.Context {
	return context.WithValue(ctx, contextKey{}, actor)
}

// FromContext returns the ActorContext attached to ctx, if any.
//
// The second return value reports whether an ActorContext was present.
// Callers must not assume that a present ActorContext is authoritative;
// trusted resolution is a future bounded slice.
func FromContext(ctx context.Context) (ActorContext, bool) {
	value, ok := ctx.Value(contextKey{}).(ActorContext)
	return value, ok
}
