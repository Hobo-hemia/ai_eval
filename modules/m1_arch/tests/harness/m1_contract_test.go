//go:build m1harness

package m1harness

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProto_BasicShape(t *testing.T) {
	raw := mustReadProto(t)
	assert.Contains(t, raw, `syntax = "proto3";`)
	assert.Contains(t, raw, "package m1.arch.v2;")
}

func TestProto_ShouldRefactorV1ServiceAndNames(t *testing.T) {
	raw := mustReadProto(t)
	assert.NotContains(t, raw, "service TicketService")
	assert.NotContains(t, raw, "rpc SubmitTicket")
	assert.NotContains(t, raw, "rpc GetTicket")
	assert.Contains(t, raw, "service")
}

func TestProto_NewRPCCountMustBeLessThanFive(t *testing.T) {
	raw := mustReadProto(t)
	re := regexp.MustCompile(`(?m)^\s*rpc\s+[A-Za-z0-9_]+\s*\(`)
	rpcs := re.FindAllString(raw, -1)
	if len(rpcs) == 0 {
		t.Fatalf("no rpc found in result proto")
	}
	assert.Less(t, len(rpcs), 5, "new rpc should be abstracted to < 5")
	assert.GreaterOrEqual(t, len(rpcs), 2, "should not over-collapse to a single rpc")
}

func TestProto_ShouldCoverPRDKeyCapabilities(t *testing.T) {
	raw := mustReadProto(t)
	// timeline
	assert.True(t, strings.Contains(raw, "timeline") || strings.Contains(raw, "TimelineEvent"))
	// collaborator
	assert.True(t, strings.Contains(raw, "collaborator") || strings.Contains(raw, "Collaborator"))
	// batch operation
	assert.True(t, strings.Contains(raw, "Batch") || strings.Contains(raw, "batch"))
	// sla risk
	assert.True(t, strings.Contains(raw, "sla") || strings.Contains(raw, "Sla") || strings.Contains(raw, "risk"))
}

func TestProto_FieldEvolutionSignals(t *testing.T) {
	raw := mustReadProto(t)
	// expected normalized source/scenario semantics.
	assert.True(t, strings.Contains(raw, "scenario") || strings.Contains(raw, "source") || strings.Contains(raw, "channel"))
}

func mustReadProto(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("m1_result.proto")
	if err != nil {
		t.Fatalf("read proto failed: %v", err)
	}
	return string(b)
}
