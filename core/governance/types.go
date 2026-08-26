// Package governance contains Axym-owned, deterministic projections of
// governed actions. These values are evidence and compliance interpretations;
// they never grant authority or execute an action.
package governance

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	proofcanon "github.com/Clyra-AI/proof/canon"
	proofsign "github.com/Clyra-AI/proof/signing"
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
	// Axes intentionally remain separate. A satisfied authorization relation
	// must not be mistaken for a precondition, confirmation, effect, or
	// outcome assertion.
	Authorization []map[string]any `json:"authorization,omitempty"`
	Preconditions []map[string]any `json:"preconditions,omitempty"`
	Confirmation  []map[string]any `json:"confirmation,omitempty"`
	Approval      []map[string]any `json:"approval,omitempty"`
	Credential    []map[string]any `json:"credential,omitempty"`
	Effect        []map[string]any `json:"effect,omitempty"`
	Compensation  []map[string]any `json:"compensation,omitempty"`
	Outcome       []map[string]any `json:"outcome,omitempty"`
	Provenance    Ref              `json:"provenance"`
	CausalRef     Ref              `json:"causal_ref,omitempty"`
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
	SchemaID      string               `json:"schema_id"`
	SchemaVersion string               `json:"schema_version"`
	PacketID      string               `json:"packet_id"`
	Contract      Contract             `json:"contract"`
	Evidence      []Evidence           `json:"evidence"`
	Completeness  string               `json:"completeness"`
	ReasonCodes   []string             `json:"reason_codes,omitempty"`
	SourceDigests []string             `json:"source_digests"`
	Digest        string               `json:"digest,omitempty"`
	Signature     *proofsign.Signature `json:"signature,omitempty"`
	AxisStates    map[string]string    `json:"axis_states,omitempty"`
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
	seenDigests := map[string]bool{contract.Provenance.Digest: true}
	for _, item := range p.Evidence {
		if item.Ref.Digest != "" && !seenDigests[item.Ref.Digest] {
			p.SourceDigests = append(p.SourceDigests, item.Ref.Digest)
			seenDigests[item.Ref.Digest] = true
		}
	}
	sort.Strings(p.SourceDigests)
	p.Completeness, p.AxisStates, p.ReasonCodes = derivePacketCompleteness(p.Evidence)
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

// derivePacketCompleteness derives each governance axis from verified
// evidence. It never treats a caller boolean or a proposal declaration as
// proof, and retains missing versus explicitly unknown states.
func derivePacketCompleteness(evidence []Evidence) (string, map[string]string, []string) {
	aliases := map[string]string{"authority": "authority", "authorization": "authority", "readiness": "readiness", "precondition": "preconditions", "preconditions": "preconditions", "confirmation": "confirmation", "credential": "credential", "delegation": "delegation", "approval": "approval", "enforcement": "enforcement", "containment": "containment", "resource": "resource_lifecycle", "resource_lifecycle": "resource_lifecycle", "compensation": "compensation", "proof": "proof", "freshness": "freshness", "correlation": "correlation", "execution": "execution", "effect": "effect", "outcome": "outcome"}
	axes := []string{"authority", "readiness", "preconditions", "confirmation", "credential", "delegation", "approval", "enforcement", "containment", "resource_lifecycle", "compensation", "proof", "freshness", "correlation", "execution", "effect", "outcome"}
	states := map[string]string{}
	terminalStates := map[string]string{}
	for _, axis := range axes {
		states[axis] = "missing"
	}
	for _, item := range evidence {
		axis := aliases[strings.ToLower(strings.TrimSpace(item.Kind))]
		if axis == "" {
			continue
		}
		state := "present"
		if item.Attributes != nil {
			switch strings.ToLower(strings.TrimSpace(item.Attributes["state"])) {
			case "unknown", "unverifiable":
				state = "unknown"
			case "gap", "failed":
				state = "gap"
			case "required":
				// A requirement without completed compensation is
				// explicit incompleteness, not proof that it ran.
				state = "gap"
			case "not_required", "completed", "succeeded":
				// This is an explicit verified disposition; it is covered
				// for axis completeness but never maps to compensated.
				state = "present"
			case "started", "requested":
				state = "unknown"
			case "partial":
				state = "unknown"
			case "blocked":
				state = "gap"
			case "recorded":
				state = "unknown"
			}
		}
		if item.Attributes != nil && item.Attributes["terminal"] == "true" {
			terminalStates[axis] = state
		} else if states[axis] == "missing" || state == "gap" || (state == "unknown" && states[axis] == "present") {
			states[axis] = state
		}
	}
	for axis, state := range terminalStates {
		states[axis] = state
	}
	reasons := make([]string, 0)
	complete, gap := true, false
	for _, axis := range axes {
		switch states[axis] {
		case "missing":
			complete = false
			reasons = append(reasons, "MISSING_"+strings.ToUpper(axis))
		case "unknown":
			complete, gap = false, true
			reasons = append(reasons, "UNKNOWN_"+strings.ToUpper(axis))
		case "gap":
			complete, gap = false, true
			reasons = append(reasons, "GAP_"+strings.ToUpper(axis))
		}
	}
	sort.Strings(reasons)
	if gap {
		return "gap", states, reasons
	}
	if complete {
		return "complete", states, reasons
	}
	return "partial", states, reasons
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
	signature := p.Signature
	p.Signature = nil
	got, err := Digest(p)
	if err != nil {
		return err
	}
	if want == "" || want != got {
		return fmt.Errorf("%s", ReasonTampered)
	}
	if signature != nil && (len(signature.SignedDigest) != 64 || signature.SignedDigest != strings.TrimPrefix(want, "sha256:")) {
		return fmt.Errorf("%s", ReasonTampered)
	}
	return nil
}

// SignPacket signs the digest after clearing its signature field. The packet
// remains portable and can be verified offline with VerifySignedPacket.
func SignPacket(p Packet, priv ed25519.PrivateKey) (Packet, error) {
	p.Signature = nil
	p.Digest = ""
	d, err := Digest(p)
	if err != nil {
		return p, err
	}
	p.Digest = d
	sig, err := proofsign.SignDigestHex(priv, strings.TrimPrefix(d, "sha256:"))
	if err != nil {
		return p, err
	}
	p.Signature = &sig
	return p, nil
}

func VerifySignedPacket(p Packet, pub ed25519.PublicKey) error {
	if err := ValidatePacket(p); err != nil {
		return err
	}
	if p.Signature == nil {
		return fmt.Errorf("%s", ReasonTampered)
	}
	if p.Signature.KeyID != proofsign.KeyID(pub) {
		return fmt.Errorf("%s", ReasonTampered)
	}
	without := p
	without.Signature = nil
	want := without.Digest
	without.Digest = ""
	got, err := Digest(without)
	if err != nil || want == "" || want != got || p.Signature.SignedDigest != strings.TrimPrefix(want, "sha256:") {
		return fmt.Errorf("%s", ReasonTampered)
	}
	ok, err := proofsign.VerifyDigestHex(pub, *p.Signature)
	if err != nil || !ok {
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
		if r.Kind == "authority" {
			bound := false
			for _, allowed := range c.AuthorityRefs {
				if r.ID == allowed.ID && r.Digest == allowed.Digest {
					bound = true
					break
				}
			}
			if !bound {
				return fmt.Errorf("authority lineage expansion")
			}
		}
	}
	return nil
}
