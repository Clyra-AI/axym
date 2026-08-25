package governance

import "sort"

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

func ProjectTimeline(contractID string, events []Event) (Timeline, error) {
	sorted := append([]Event(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].OccurredAt == sorted[j].OccurredAt {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].OccurredAt < sorted[j].OccurredAt
	})
	state, err := Reduce(contractID, sorted)
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
