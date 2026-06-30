package export

import (
	"testing"
)

func TestRegistryExportFormats(t *testing.T) {
	snap := &Snapshot{
		InvestigationID: "inv-1",
		Question:        "Why did checkout fail?",
		State:           "collecting_evidence",
		Confidence:      42,
		Graph: GraphView{
			Nodes: []GraphNode{{ID: "ev:1", Type: "evidence", Label: "error log"}},
			Edges: []GraphEdge{{From: "ev:1", To: "hyp:1", Type: "supports"}},
		},
		Hypotheses: []Hypothesis{{ID: "hypothesis-deployment-caused", Statement: "Bad deploy", Confidence: 55}},
	}
	reg := NewRegistry()
	for _, format := range []Format{FormatMarkdown, FormatJSON, FormatMermaid, FormatGraphML, FormatPlantUML} {
		out, err := reg.Export(format, snap)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if len(out) == 0 {
			t.Fatalf("%s: empty output", format)
		}
	}
}
