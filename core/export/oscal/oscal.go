package oscal

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
	bundleschema "github.com/Clyra-AI/axym/schemas/v1/bundle"
)

const (
	FixedTimestamp = "2000-01-01T00:00:00Z"
	Version        = "1.1"
)

type ComponentDefinitionDocument struct {
	ComponentDefinition ComponentDefinition `json:"component-definition"`
}

type ComponentDefinition struct {
	UUID       string      `json:"uuid"`
	Metadata   Metadata    `json:"metadata"`
	Components []Component `json:"components"`
}

type Metadata struct {
	Title        string `json:"title"`
	Version      string `json:"version"`
	LastModified string `json:"last-modified"`
}

type Component struct {
	UUID                   string                  `json:"uuid"`
	Type                   string                  `json:"type"`
	Title                  string                  `json:"title"`
	Description            string                  `json:"description,omitempty"`
	ControlImplementations []ControlImplementation `json:"control-implementations"`
}

type ControlImplementation struct {
	Source                  string                   `json:"source"`
	Description             string                   `json:"description,omitempty"`
	ImplementedRequirements []ImplementedRequirement `json:"implemented-requirements"`
}

type ImplementedRequirement struct {
	ControlID   string `json:"control-id"`
	Description string `json:"description"`
}

func Build(auditName string, report coverage.Report) ComponentDefinitionDocument {
	implementations := make([]ControlImplementation, 0, len(report.Frameworks))
	for _, fw := range report.Frameworks {
		requirements := make([]ImplementedRequirement, 0, len(fw.Controls))
		for _, control := range fw.Controls {
			requirements = append(requirements, ImplementedRequirement{
				ControlID:   control.ControlID,
				Description: fmt.Sprintf("status=%s evidence=%d reason_codes=%v", control.Status, control.EvidenceCount, control.ReasonCodes),
			})
		}
		sort.Slice(requirements, func(i, j int) bool {
			return requirements[i].ControlID < requirements[j].ControlID
		})
		implementations = append(implementations, ControlImplementation{
			Source:                  fw.FrameworkID,
			Description:             "Deterministic Axym mapping from local proof records.",
			ImplementedRequirements: requirements,
		})
	}
	sort.Slice(implementations, func(i, j int) bool {
		return implementations[i].Source < implementations[j].Source
	})

	title := fmt.Sprintf("Axym Component Definition (%s)", auditName)
	return ComponentDefinitionDocument{
		ComponentDefinition: ComponentDefinition{
			UUID: "axym-component-definition",
			Metadata: Metadata{
				Title:        title,
				Version:      Version,
				LastModified: FixedTimestamp,
			},
			Components: []Component{
				{
					UUID:                   "axym-cli",
					Type:                   "software",
					Title:                  "Axym CLI",
					Description:            "Deterministic evidence collection, compliance mapping, and bundle generation.",
					ControlImplementations: implementations,
				},
			},
		},
	}
}

func Marshal(doc ComponentDefinitionDocument) ([]byte, error) {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := bundleschema.ValidateOSCAL(raw); err != nil {
		return nil, err
	}
	return raw, nil
}
