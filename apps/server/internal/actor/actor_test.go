package actor

import (
	"context"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestFromContextReturnsFalseWhenNoActor(t *testing.T) {
	t.Parallel()

	if got, ok := FromContext(context.Background()); ok {
		t.Fatalf("FromContext on empty context = (%#v, true), want (zero, false)", got)
	}
}

func TestWithActorRoundTrip(t *testing.T) {
	t.Parallel()

	want := ActorContext{
		Actor: spine.ActorRef{
			Kind:        "user",
			ID:          "dev_1",
			DisplayName: "Developer",
		},
		Source: SourceDevHeader,
	}

	ctx := WithActor(context.Background(), want)

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("FromContext after WithActor returned ok = false, want true")
	}
	if got != want {
		t.Fatalf("FromContext = %#v, want %#v", got, want)
	}
}

func TestWithActorPreservesSource(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		source Source
	}{
		{"unknown", SourceUnknown},
		{"dev_header", SourceDevHeader},
		{"payload_compat", SourcePayloadCompat},
		{"service", SourceService},
		{"system", SourceSystem},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := ActorContext{
				Actor:  spine.ActorRef{Kind: "user", ID: "dev_1"},
				Source: tc.source,
			}
			ctx := WithActor(context.Background(), input)

			got, ok := FromContext(ctx)
			if !ok {
				t.Fatal("FromContext returned ok = false, want true")
			}
			if got.Source != tc.source {
				t.Fatalf("Source = %q, want %q", got.Source, tc.source)
			}
		})
	}
}
