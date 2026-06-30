// Package export provides investigation exporters decoupled from the runtime.
//
// Build a Snapshot from session state (or use export.FromSession in internal
// wiring), then pass it to any Exporter. Exporters do not mutate snapshots.
//
// Supported formats: Markdown, JSON, Mermaid, GraphML, PlantUML.
package export

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Format identifies an export encoding.
type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatJSON     Format = "json"
	FormatMermaid  Format = "mermaid"
	FormatGraphML  Format = "graphml"
	FormatPlantUML Format = "plantuml"
)

// Snapshot is a portable investigation view for exporters.
type Snapshot struct {
	InvestigationID string       `json:"investigation_id"`
	Question        string       `json:"question"`
	Service         string       `json:"service,omitempty"`
	State           string       `json:"state"`
	Confidence      float64      `json:"confidence"`
	Evidence        []Evidence   `json:"evidence"`
	Hypotheses      []Hypothesis `json:"hypotheses"`
	Graph           GraphView    `json:"graph"`
	Report          *Report      `json:"report,omitempty"`
	ExportedAt      time.Time    `json:"exported_at"`
}

// Evidence is a vendor-neutral observation.
type Evidence struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Category  string    `json:"category"`
	Entity    string    `json:"entity,omitempty"`
	Summary   string    `json:"summary"`
}

// Hypothesis is one competing explanation.
type Hypothesis struct {
	ID         string  `json:"id"`
	Statement  string  `json:"statement"`
	Confidence float64 `json:"confidence"`
	Status     string  `json:"status"`
	Rationale  string  `json:"rationale,omitempty"`
}

// GraphNode is a node in the investigation graph.
type GraphNode struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

// GraphEdge connects graph nodes.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// GraphView is a serializable graph.
type GraphView struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// Report is the final investigation deliverable.
type Report struct {
	ExecutiveSummary string   `json:"executive_summary"`
	Confidence       float64  `json:"confidence"`
	Postmortem       string   `json:"postmortem,omitempty"`
	Recommendations  []string `json:"recommendations,omitempty"`
}

// Exporter writes a snapshot in a specific format.
type Exporter interface {
	Format() Format
	Export(snap *Snapshot) ([]byte, error)
}

// Registry holds exporters by format.
type Registry struct {
	exporters map[Format]Exporter
}

// NewRegistry returns a registry with built-in exporters.
func NewRegistry() *Registry {
	r := &Registry{exporters: map[Format]Exporter{}}
	for _, e := range []Exporter{
		MarkdownExporter{},
		JSONExporter{},
		MermaidExporter{},
		GraphMLExporter{},
		PlantUMLExporter{},
	} {
		r.exporters[e.Format()] = e
	}
	return r
}

// Register adds or replaces an exporter for its format.
func (r *Registry) Register(e Exporter) {
	if e == nil {
		return
	}
	r.exporters[e.Format()] = e
}

// Export renders a snapshot in the requested format.
func (r *Registry) Export(format Format, snap *Snapshot) ([]byte, error) {
	e, ok := r.exporters[format]
	if !ok {
		return nil, fmt.Errorf("unsupported export format %q", format)
	}
	if snap == nil {
		return nil, fmt.Errorf("nil snapshot")
	}
	return e.Export(snap)
}

// MarkdownExporter renders a human-readable investigation summary.
type MarkdownExporter struct{}

func (MarkdownExporter) Format() Format { return FormatMarkdown }

func (MarkdownExporter) Export(snap *Snapshot) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# Investigation %s\n\n", snap.InvestigationID)
	fmt.Fprintf(&b, "**Question:** %s\n\n", snap.Question)
	fmt.Fprintf(&b, "**State:** %s | **Confidence:** %.1f%%\n\n", snap.State, snap.Confidence)
	if snap.Report != nil {
		fmt.Fprintf(&b, "## Executive summary\n\n%s\n\n", snap.Report.ExecutiveSummary)
		if len(snap.Report.Recommendations) > 0 {
			b.WriteString("## Recommendations\n\n")
			for _, rec := range snap.Report.Recommendations {
				fmt.Fprintf(&b, "- %s\n", rec)
			}
			b.WriteString("\n")
		}
	}
	if len(snap.Hypotheses) > 0 {
		b.WriteString("## Hypotheses\n\n")
		for _, h := range snap.Hypotheses {
			fmt.Fprintf(&b, "- **%s** (%.1f%%): %s\n", h.ID, h.Confidence, h.Statement)
		}
		b.WriteString("\n")
	}
	if len(snap.Evidence) > 0 {
		b.WriteString("## Evidence\n\n")
		for _, e := range snap.Evidence {
			fmt.Fprintf(&b, "- `%s` [%s] %s\n", e.ID, e.Category, e.Summary)
		}
	}
	return []byte(b.String()), nil
}

// JSONExporter renders the full snapshot as JSON.
type JSONExporter struct{}

func (JSONExporter) Format() Format { return FormatJSON }

func (JSONExporter) Export(snap *Snapshot) ([]byte, error) {
	return json.MarshalIndent(snap, "", "  ")
}

// MermaidExporter renders the graph as a Mermaid flowchart.
type MermaidExporter struct{}

func (MermaidExporter) Format() Format { return FormatMermaid }

func (MermaidExporter) Export(snap *Snapshot) ([]byte, error) {
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	for _, n := range snap.Graph.Nodes {
		safe := strings.ReplaceAll(n.Label, "\"", "'")
		fmt.Fprintf(&b, "  %s[\"%s\"]\n", sanitizeID(n.ID), safe)
	}
	for _, e := range snap.Graph.Edges {
		fmt.Fprintf(&b, "  %s -->|%s| %s\n", sanitizeID(e.From), e.Type, sanitizeID(e.To))
	}
	return []byte(b.String()), nil
}

// GraphMLExporter renders the graph as GraphML XML.
type GraphMLExporter struct{}

func (GraphMLExporter) Format() Format { return FormatGraphML }

func (GraphMLExporter) Export(snap *Snapshot) ([]byte, error) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<graphml xmlns="http://graphml.graphdrawing.org/xmlns">` + "\n")
	b.WriteString(`<graph edgedefault="directed">` + "\n")
	for _, n := range snap.Graph.Nodes {
		fmt.Fprintf(&b, `<node id="%s"><data key="label">%s</data></node>`+"\n", xmlEscape(n.ID), xmlEscape(n.Label))
	}
	for i, e := range snap.Graph.Edges {
		fmt.Fprintf(&b, `<edge id="e%d" source="%s" target="%s"><data key="type">%s</data></edge>`+"\n",
			i, xmlEscape(e.From), xmlEscape(e.To), xmlEscape(e.Type))
	}
	b.WriteString("</graph>\n</graphml>\n")
	return []byte(b.String()), nil
}

// PlantUMLExporter renders the graph as PlantUML.
type PlantUMLExporter struct{}

func (PlantUMLExporter) Format() Format { return FormatPlantUML }

func (PlantUMLExporter) Export(snap *Snapshot) ([]byte, error) {
	var b strings.Builder
	b.WriteString("@startuml\n")
	for _, n := range snap.Graph.Nodes {
		fmt.Fprintf(&b, "component \"%s\" as %s\n", strings.ReplaceAll(n.Label, "\"", "'"), sanitizeID(n.ID))
	}
	for _, e := range snap.Graph.Edges {
		fmt.Fprintf(&b, "%s --> %s : %s\n", sanitizeID(e.From), sanitizeID(e.To), e.Type)
	}
	b.WriteString("@enduml\n")
	return []byte(b.String()), nil
}

func sanitizeID(id string) string {
	return strings.NewReplacer(":", "_", "-", "_", ".", "_").Replace(id)
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
