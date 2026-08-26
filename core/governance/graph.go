package governance

import (
	"sort"
	"strings"
)

type Timeline struct {
	ContractID    string   `json:"contract_id"`
	Events        []Event  `json:"events"`
	State         State    `json:"state"`
	SourceDigests []string `json:"source_digests"`
}
type GraphNode struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Digest string `json:"digest,omitempty"`
}
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}
type Graph struct {
	Nodes       []GraphNode `json:"nodes"`
	Edges       []GraphEdge `json:"edges"`
	ReplayState State       `json:"replay_state"`
}

// EvidenceEventKind is the single semantic mapping from packet evidence to
// timeline events. Callers must not infer lifecycle outcomes from the axis
// name alone: failed execution, rejected effects, and unresolved containment
// are materially different audit states.
func EvidenceEventKind(evidenceKind, state string) string {
	kind := strings.ToLower(strings.TrimSpace(evidenceKind))
	state = strings.ToLower(strings.TrimSpace(state))
	switch kind {
	case "proposal":
		return "proposed"
	case "approval":
		return "approved"
	case "activation", "policy_enforcement", "enforcement":
		return "activated"
	case "tool_invocation", "execution_started":
		return "execution_started"
	case "execution", "execution_failed":
		if state == "failed" || state == "gap" || state == "rejected" {
			return "execution_failed"
		}
		if state == "blocked" {
			return "execution_blocked"
		}
		if state == "succeeded" || state == "completed" {
			return "execution_succeeded"
		}
		return "execution_started"
	case "test_result", "effect_validated", "effect":
		if state == "recorded" {
			return "effect_recorded"
		}
		if state == "rejected" || state == "failed" || state == "gap" {
			return "effect_rejected"
		}
		if state == "unknown" || state == "unresolved" || state == "started" || state == "requested" || state == "" {
			return "effect_unresolved"
		}
		return "effect_validated"
	case "contained", "containment":
		if state == "partial" {
			return "containment_partial"
		}
		if state == "requested" || state == "started" {
			return "containment_requested"
		}
		if state == "unresolved" || state == "unknown" || state == "gap" {
			return "containment_unresolved"
		}
		return "contained"
	case "compensation":
		switch state {
		case "required":
			return "compensation_required"
		case "not_required":
			return "compensation_not_required"
		case "failed", "rejected", "gap":
			return "compensation_failed"
		case "unknown", "unresolved":
			return "compensation_unresolved"
		case "started", "requested":
			return "compensation_started"
		case "completed", "succeeded", "present":
			return "compensated"
		default:
			return "compensation_unresolved"
		}
	default:
		return ""
	}
}

// EventsFromPacket converts governed packet evidence into reducer events while
// preserving failed/unknown semantic states.
func EventsFromPacket(packet Packet) []Event {
	events := make([]Event, 0, len(packet.Evidence))
	for _, evidence := range packet.Evidence {
		state := ""
		eventID := evidence.Ref.ID
		sourceDigest := evidence.Ref.Digest
		if evidence.Attributes != nil {
			state = evidence.Attributes["state"]
			if evidence.Attributes["event_id"] != "" {
				eventID = evidence.Attributes["event_id"]
			}
			if evidence.Attributes["source_digest"] != "" {
				sourceDigest = evidence.Attributes["source_digest"]
			}
		}
		kind := EvidenceEventKind(evidence.Kind, state)
		if kind == "" {
			continue
		}
		events = append(events, Event{ID: eventID, ContractRef: evidence.ContractRef, Kind: kind, OccurredAt: evidence.OccurredAt, Status: state, SourceDigest: sourceDigest})
	}
	return events
}

func ProjectTimeline(contractID string, events []Event) (Timeline, error) {
	sorted := append([]Event(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return eventBefore(sorted[i], sorted[j])
	})
	var state State
	var err error
	if len(sorted) > 0 && validRef(sorted[0].ContractRef) {
		state, err = ReduceVerified(contractID, sorted)
	} else {
		// Legacy in-memory callers may provide only IDs. Production ingest
		// always supplies complete digest-bound refs and takes the strict path.
		state, err = Reduce(contractID, sorted)
	}
	if err != nil {
		return Timeline{}, err
	}
	return Timeline{ContractID: contractID, Events: sorted, State: state, SourceDigests: append([]string(nil), state.SourceDigests...)}, nil
}
func ProjectGraph(t Timeline) (Graph, error) {
	g := Graph{Nodes: []GraphNode{{ID: t.ContractID, Kind: "action_contract"}}, ReplayState: t.State}
	for _, e := range t.Events {
		g.Nodes = append(g.Nodes, GraphNode{ID: e.ID, Kind: e.Kind, Digest: e.SourceDigest})
		g.Edges = append(g.Edges, GraphEdge{From: t.ContractID, To: e.ID, Kind: "has_event"})
	}
	sort.Slice(g.Nodes, func(i, j int) bool {
		if g.Nodes[i].Kind == g.Nodes[j].Kind {
			return g.Nodes[i].ID < g.Nodes[j].ID
		}
		return g.Nodes[i].Kind < g.Nodes[j].Kind
	})
	sort.Slice(g.Edges, func(i, j int) bool {
		if g.Edges[i].From == g.Edges[j].From {
			return g.Edges[i].To < g.Edges[j].To
		}
		return g.Edges[i].From < g.Edges[j].From
	})
	return g, nil
}
