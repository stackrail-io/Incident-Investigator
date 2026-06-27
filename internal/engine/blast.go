package engine

import (
	"fmt"
	"sort"

	"github.com/stackrail/incident-investigator/internal/model"
)

// BlastRadiusEstimator estimates the scope of impact.
type BlastRadiusEstimator interface {
	Estimate(s *model.Session, sig Signals) model.BlastRadius
}

// HeuristicBlastRadiusEstimator derives impact from entities seen in evidence
// and well-known payload keys (region, customer, endpoint/api).
type HeuristicBlastRadiusEstimator struct{}

// NewHeuristicBlastRadiusEstimator returns the default estimator.
func NewHeuristicBlastRadiusEstimator() *HeuristicBlastRadiusEstimator {
	return &HeuristicBlastRadiusEstimator{}
}

var blastPayloadKeys = map[string][]string{
	"region":   {"region", "az", "zone", "datacenter"},
	"customer": {"customer", "customers", "tenant", "account", "customer_impact"},
	"api":      {"api", "endpoint", "route", "path", "operation"},
}

// Estimate implements BlastRadiusEstimator.
func (b *HeuristicBlastRadiusEstimator) Estimate(s *model.Session, sig Signals) model.BlastRadius {
	var br model.BlastRadius

	// Services: the primary service plus every distinct entity observed.
	svcConf := map[string]float64{}
	if s.Service != "" {
		svcConf[s.Service] = 90
	}
	for name, count := range sig.Entities {
		conf := 50.0 + float64(count)*10
		if name == s.Service {
			conf = 90
		}
		if conf > 95 {
			conf = 95
		}
		if existing, ok := svcConf[name]; !ok || conf > existing {
			svcConf[name] = conf
		}
	}
	br.Services = toScopedItems(svcConf)

	regions := collectPayloadValues(s, blastPayloadKeys["region"])
	customers := collectPayloadValues(s, blastPayloadKeys["customer"])
	apis := collectPayloadValues(s, blastPayloadKeys["api"])

	br.Regions = toScopedItems(regions)
	br.Customers = toScopedItems(customers)
	br.APIs = toScopedItems(apis)

	return br
}

// collectPayloadValues scans evidence payloads for the given keys and counts
// distinct values, mapping frequency to a confidence score.
func collectPayloadValues(s *model.Session, keys []string) map[string]float64 {
	counts := map[string]int{}
	for _, e := range s.Evidence {
		for _, k := range keys {
			if v, ok := e.Payload[k]; ok {
				for _, val := range flattenValue(v) {
					if val != "" {
						counts[val]++
					}
				}
			}
		}
	}
	out := map[string]float64{}
	for name, c := range counts {
		conf := 55.0 + float64(c)*10
		if conf > 95 {
			conf = 95
		}
		out[name] = conf
	}
	return out
}

func flattenValue(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []string:
		return t
	case []any:
		var out []string
		for _, item := range t {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return []string{fmt.Sprint(t)}
	}
}

func toScopedItems(m map[string]float64) []model.ScopedItem {
	out := make([]model.ScopedItem, 0, len(m))
	for name, conf := range m {
		out = append(out, model.ScopedItem{Name: name, Confidence: round1(conf)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		return out[i].Name < out[j].Name
	})
	if out == nil {
		return []model.ScopedItem{}
	}
	return out
}
