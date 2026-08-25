// Package governance contains Axym-owned, deterministic projections of
// governed actions. These values are evidence and compliance interpretations;
// they never grant authority or execute an action.
package governance

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	proofcanon "github.com/Clyra-AI/proof/canon"
)

const RegisterSchemaID = "https://axym.dev/schemas/v1/governance/action-contract-register.schema.json"
const PacketSchemaID = "https://axym.dev/schemas/v1/governance/action-contract-evidence-packet.schema.json"
const SchemaVersion = "v1"

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type Ref struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Digest        string `json:"digest"`
	Source        string `json:"source"`
	SourceProduct string `json:"source_product"`
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
}
type Contract struct {
	ID            string `json:"id"`
	FamilyID      string `json:"family_id"`
	Revision      int    `json:"revision"`
	Action        string `json:"action"`
	Target        string `json:"target"`
	Environment   string `json:"environment"`
	PolicyDigest  string `json:"policy_digest"`
	Owner         string `json:"owner"`
	AuthorityRefs []Ref  `json:"authority_refs,omitempty"`
	Provenance    Ref    `json:"provenance"`
}
type Register struct {
	SchemaID      string     `json:"schema_id"`
	SchemaVersion string     `json:"schema_version"`
	Contracts     []Contract `json:"contracts"`
	SourceDigest  string     `json:"source_digest,omitempty"`
}
type Evidence struct {
	Kind        string            `json:"kind"`
	Ref         Ref               `json:"ref"`
	OccurredAt  string            `json:"occurred_at"`
	ContractRef Ref               `json:"contract_ref"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Provenance  Ref               `json:"provenance"`
}
type Packet struct {
	SchemaID      string     `json:"schema_id"`
	SchemaVersion string     `json:"schema_version"`
	PacketID      string     `json:"packet_id"`
	Contract      Contract   `json:"contract"`
	Evidence      []Evidence `json:"evidence"`
	Completeness  string     `json:"completeness"`
	ReasonCodes   []string   `json:"reason_codes,omitempty"`
	SourceDigests []string   `json:"source_digests"`
	Digest        string     `json:"digest,omitempty"`
}

func Digest(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	d, err := proofcanon.DigestJCS(raw)
	if err != nil {
		return "", err
	}
	return "sha256:" + d, nil
}

func ValidateRegister(r Register) error {
	if r.SchemaID != RegisterSchemaID {
		return fmt.Errorf("unsupported register schema_id %q", r.SchemaID)
	}
	if r.SchemaVersion == "" {
		r.SchemaVersion = SchemaVersion
	}
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported register schema %q", r.SchemaVersion)
	}
	seen := map[string]bool{}
	for _, c := range r.Contracts {
		if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.FamilyID) == "" || c.Revision < 1 || strings.TrimSpace(c.Action) == "" || strings.TrimSpace(c.Target) == "" {
			return fmt.Errorf("invalid contract %q", c.ID)
		}
		if seen[c.ID] {
			return fmt.Errorf("duplicate contract %q", c.ID)
		}
		seen[c.ID] = true
		if !validRef(c.Provenance) {
			return fmt.Errorf("contract %q provenance required", c.ID)
		}
	}
	return nil
}

func ValidatePacket(p Packet) error {
	if p.SchemaID != PacketSchemaID {
		return fmt.Errorf("unsupported packet schema_id %q", p.SchemaID)
	}
	if p.SchemaVersion != "" && p.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported packet schema %q", p.SchemaVersion)
	}
	if strings.TrimSpace(p.PacketID) == "" {
		return fmt.Errorf("packet_id is required")
	}
	for _, d := range p.SourceDigests {
		if !digestPattern.MatchString(d) {
			return fmt.Errorf("invalid source digest")
		}
	}
	if err := validateContract(p.Contract); err != nil {
		return err
	}
	if len(p.Evidence) == 0 {
		return fmt.Errorf("evidence is required")
	}
	for _, e := range p.Evidence {
		if e.Kind == "" || !validRef(e.Ref) || e.ContractRef.ID != p.Contract.ID || !validRef(e.ContractRef) {
			return fmt.Errorf("evidence is not bound to contract")
		}
		if !validRef(e.Provenance) {
			return fmt.Errorf("evidence provenance is required")
		}
		if _, err := time.Parse(time.RFC3339Nano, e.OccurredAt); err != nil {
			return fmt.Errorf("evidence timestamp is invalid")
		}
	}
	return nil
}

func NormalizeRegister(r Register) (Register, error) {
	if err := ValidateRegister(r); err != nil {
		return Register{}, err
	}
	r.SchemaID, r.SchemaVersion = RegisterSchemaID, SchemaVersion
	sort.Slice(r.Contracts, func(i, j int) bool { return r.Contracts[i].ID < r.Contracts[j].ID })
	return r, nil
}
func BuildPacket(contract Contract, evidence []Evidence) (Packet, error) {
	p := Packet{SchemaID: PacketSchemaID, SchemaVersion: SchemaVersion, PacketID: contract.ID + ":" + fmt.Sprint(contract.Revision), Contract: contract, Evidence: append([]Evidence(nil), evidence...), SourceDigests: []string{contract.Provenance.Digest}}
	sort.Slice(p.Evidence, func(i, j int) bool { return p.Evidence[i].Ref.ID < p.Evidence[j].Ref.ID })
	p.Completeness = "partial"
	kinds := map[string]bool{}
	for _, e := range p.Evidence {
		kinds[e.Kind] = true
	}
	if kinds["approval"] && (kinds["execution"] || kinds["execution_started"]) && (kinds["effect"] || kinds["effect_validated"]) {
		p.Completeness = "complete"
	}
	if err := ValidatePacket(p); err != nil {
		return Packet{}, err
	}
	d, err := Digest(p)
	if err != nil {
		return Packet{}, err
	}
	p.Digest = d
	return p, nil
}

func validRef(r Ref) bool {
	return r.ID != "" && r.Kind != "" && digestPattern.MatchString(r.Digest) && r.Source != "" && r.SourceProduct != "" && r.SchemaID != "" && r.SchemaVersion != ""
}
func validateContract(c Contract) error {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.FamilyID) == "" || c.Revision < 1 || strings.TrimSpace(c.Action) == "" || strings.TrimSpace(c.Target) == "" {
		return fmt.Errorf("invalid contract %q", c.ID)
	}
	if !validRef(c.Provenance) {
		return fmt.Errorf("contract %q provenance required", c.ID)
	}
	return nil
}

func VerifyPacket(p Packet) error {
	if err := ValidatePacket(p); err != nil {
		return err
	}
	if !digestPattern.MatchString(p.Digest) {
		return fmt.Errorf("%s", ReasonTampered)
	}
	want := p.Digest
	p.Digest = ""
	got, err := Digest(p)
	if err != nil {
		return err
	}
	if want == "" || want != got {
		return fmt.Errorf("%s", ReasonTampered)
	}
	return nil
}

// VerifyLineage confirms references remain within the declared contract. It
// deliberately rejects authority expansion (new target, environment, policy,
// or unbound authority) instead of inferring permission from evidence.
func VerifyLineage(c Contract, refs []Ref) error {
	for _, r := range refs {
		if !validRef(r) {
			return fmt.Errorf("incomplete lineage reference")
		}
		if r.Kind == "target" && r.ID != c.Target {
			return fmt.Errorf("target authority expansion")
		}
		if r.Kind == "environment" && r.ID != c.Environment {
			return fmt.Errorf("environment authority expansion")
		}
		if r.Kind == "policy" && r.Digest != c.PolicyDigest {
			return fmt.Errorf("policy authority expansion")
		}
	}
	return nil
}
