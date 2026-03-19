package record

import "github.com/Clyra-AI/proof"

func canonicalizeEventAndMetadata(event map[string]any, metadata map[string]any) (map[string]any, map[string]any) {
	if event == nil {
		event = map[string]any{}
	}
	return event, canonicalizeOptionalMap(metadata)
}

func CanonicalizeProofRecord(in *proof.Record) *proof.Record {
	if in == nil {
		return nil
	}

	out := proof.Record(*in)
	if in.Event != nil {
		out.Event = cloneMap(in.Event)
	}
	if in.Metadata != nil {
		out.Metadata = cloneMap(in.Metadata)
	}
	out.Metadata = canonicalizeOptionalMap(out.Metadata)
	return &out
}

func canonicalizeOptionalMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	return in
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
