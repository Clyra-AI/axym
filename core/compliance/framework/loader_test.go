package framework

import (
	"reflect"
	"testing"

	proofframework "github.com/Clyra-AI/proof/core/framework"
)

func TestLoadManyDeterministicAndStrict(t *testing.T) {
	t.Parallel()

	frameworks, err := LoadMany([]string{"soc2", "eu-ai-act", "soc2"})
	if err != nil {
		t.Fatalf("LoadMany: %v", err)
	}
	if len(frameworks) != 2 {
		t.Fatalf("framework count mismatch: %d", len(frameworks))
	}
	if frameworks[0].ID != "eu-ai-act" || frameworks[1].ID != "soc2" {
		t.Fatalf("framework order mismatch: %+v", frameworks)
	}
	if len(frameworks[0].Controls) == 0 {
		t.Fatalf("expected controls for %s", frameworks[0].ID)
	}
	for _, definition := range frameworks {
		for _, control := range definition.Controls {
			if len(control.EvidenceSets) == 0 {
				if len(control.RequiredRecordTypes) == 0 {
					t.Fatalf("legacy control %s/%s dropped required record types", definition.ID, control.ID)
				}
				continue
			}
			for _, set := range control.EvidenceSets {
				if len(set.RequiredRecordTypes) == 0 {
					t.Fatalf("control %s/%s evidence set %s dropped required record types", definition.ID, control.ID, set.ID)
				}
				if len(set.RequiredFields) == 0 {
					t.Fatalf("control %s/%s evidence set %s dropped required fields", definition.ID, control.ID, set.ID)
				}
				if set.MinimumFrequency == "" {
					t.Fatalf("control %s/%s evidence set %s dropped minimum frequency", definition.ID, control.ID, set.ID)
				}
			}
		}
	}
}

func TestFlattenControlsPreservesLegacyRequirements(t *testing.T) {
	t.Parallel()

	input := proofframework.Control{
		ID:                  "legacy-control",
		Title:               "Legacy control",
		RequiredRecordTypes: []string{"risk_assessment", "approval"},
		RequiredFields:      []string{"timestamp", "record_id"},
		MinimumFrequency:    "quarterly",
	}
	var controls []Control
	flattenControls("fixture", []proofframework.Control{input}, &controls)

	if len(controls) != 1 {
		t.Fatalf("control count mismatch: got %d want 1", len(controls))
	}
	if got, want := controls[0].RequiredRecordTypes, []string{"approval", "risk_assessment"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("required record types mismatch: got %v want %v", got, want)
	}
	if got, want := controls[0].RequiredFields, []string{"record_id", "timestamp"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("required fields mismatch: got %v want %v", got, want)
	}
	if got, want := controls[0].MinimumFrequency, "quarterly"; got != want {
		t.Fatalf("minimum frequency mismatch: got %q want %q", got, want)
	}
}

func TestFlattenControlsAdaptsEvidenceSetRequirements(t *testing.T) {
	t.Parallel()

	input := proofframework.Control{
		ID:    "evidence-set-control",
		Title: "Evidence-set control",
		EvidenceSets: []proofframework.EvidenceSet{
			{
				ID:                  "runtime",
				SourceProducts:      []string{"gait"},
				RequiredRecordTypes: []string{"decision", "approval"},
				RequiredFields:      []string{"record_id", "event"},
				MinimumFrequency:    "continuous",
			},
			{
				ID:                  "discovery",
				SourceProducts:      []string{"wrkr", "axym"},
				RequiredRecordTypes: []string{"scan_finding", "decision"},
				RequiredFields:      []string{"source_product", "record_id"},
				MinimumFrequency:    "quarterly",
			},
		},
	}
	var controls []Control
	flattenControls("fixture", []proofframework.Control{input}, &controls)

	if len(controls) != 1 {
		t.Fatalf("control count mismatch: got %d want 1", len(controls))
	}
	if len(controls[0].RequiredRecordTypes) != 0 {
		t.Fatalf("evidence-set control must not flatten alternatives into legacy requirements: %v", controls[0].RequiredRecordTypes)
	}
	if len(controls[0].EvidenceSets) != 2 {
		t.Fatalf("evidence set count mismatch: got %d want 2", len(controls[0].EvidenceSets))
	}
	if got, want := controls[0].EvidenceSets[0].ID, "discovery"; got != want {
		t.Fatalf("evidence set order mismatch: got %q want %q", got, want)
	}
	if got, want := controls[0].EvidenceSets[0].RequiredRecordTypes, []string{"decision", "scan_finding"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("discovery record types mismatch: got %v want %v", got, want)
	}
	if got, want := controls[0].EvidenceSets[0].SourceProducts, []string{"axym", "wrkr"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("discovery source products mismatch: got %v want %v", got, want)
	}
	if got, want := controls[0].EvidenceSets[1].RequiredRecordTypes, []string{"approval", "decision"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runtime record types mismatch: got %v want %v", got, want)
	}
	if got, want := controls[0].EvidenceSets[1].RequiredFields, []string{"event", "record_id"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runtime required fields mismatch: got %v want %v", got, want)
	}
	if got, want := controls[0].EvidenceSets[1].MinimumFrequency, "continuous"; got != want {
		t.Fatalf("runtime minimum frequency mismatch: got %q want %q", got, want)
	}
}

func TestLoadManyMissingFramework(t *testing.T) {
	t.Parallel()

	_, err := LoadMany([]string{"does-not-exist"})
	if err == nil {
		t.Fatalf("expected error")
	}
	loadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("unexpected error type: %T", err)
	}
	if loadErr.ReasonCode != ReasonFrameworkLoad {
		t.Fatalf("reason mismatch: %s", loadErr.ReasonCode)
	}
}
