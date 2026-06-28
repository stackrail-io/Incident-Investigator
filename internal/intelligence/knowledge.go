package intelligence

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// ExtractLessons derives reusable lessons from similar archived investigations.
func ExtractLessons(results []model.SimilarityResult, archive InvestigationArchive) []model.LessonLearned {
	type key struct {
		root  string
		cat   model.Category
	}
	counts := map[key]int{}
	for _, r := range results {
		snap, err := archive.Find(r.InvestigationID)
		if err != nil {
			continue
		}
		for _, sum := range snap.EvidenceSummary {
			counts[key{root: snap.RootCause, cat: sum.Category}]++
		}
	}

	type lesson struct {
		l model.LessonLearned
	}
	var lessons []lesson
	seen := map[string]bool{}
	for k, n := range counts {
		if n < 1 || k.root == "" {
			continue
		}
		id := k.root + "-" + string(k.cat)
		if seen[id] {
			continue
		}
		seen[id] = true
		lessons = append(lessons, lesson{model.LessonLearned{
			ID: id,
			Summary: fmt.Sprintf("%s incidents usually require %s before confidence exceeds 80%%.",
				humanHypothesis(k.root), string(k.cat)),
			RequiredEvidence:    []model.Category{k.cat},
			ConfidenceThreshold: 80,
			Occurrences:         n,
		}})
	}
	sort.Slice(lessons, func(i, j int) bool { return lessons[i].l.Occurrences > lessons[j].l.Occurrences })
	out := make([]model.LessonLearned, 0, len(lessons))
	for _, l := range lessons {
		out = append(out, l.l)
	}
	if out == nil {
		out = []model.LessonLearned{}
	}
	return out
}

// BuildRecommendations assembles intelligence outputs for the runtime.
func BuildRecommendations(results []model.SimilarityResult, patterns []model.SuggestedPattern, lessons []model.LessonLearned, archive InvestigationArchive) *model.InvestigationRecommendations {
	rec := &model.InvestigationRecommendations{
		SimilarInvestigations: results,
		Patterns:              patterns,
		Lessons:               lessons,
	}
	rootCauseCounts := map[string]int{}
	missingCats := map[model.Category]int{}
	questionSet := map[string]model.QuestionTemplate{}

	for _, r := range results {
		if r.RootCause != "" {
			rootCauseCounts[r.RootCause]++
		}
		snap, err := archive.Find(r.InvestigationID)
		if err != nil {
			continue
		}
		for _, q := range snap.Questions {
			if q.Title != "" {
				questionSet[q.Title] = model.QuestionTemplate{Text: q.Title, Categories: q.RequiredEvidence}
			}
		}
	}

	for rc, n := range rootCauseCounts {
		if n >= 1 {
			rec.TypicalRootCauses = append(rec.TypicalRootCauses, rc)
		}
	}
	sort.Strings(rec.TypicalRootCauses)

	for _, pat := range patterns {
		for _, q := range pat.RecommendedQuestions {
			questionSet[q.Text] = q
		}
		for _, c := range pat.EvidenceCategories {
			missingCats[c]++
		}
	}
	for _, l := range lessons {
		for _, c := range l.RequiredEvidence {
			missingCats[c]++
		}
	}

	type catCount struct {
		c model.Category
		n int
	}
	var cats []catCount
	for c, n := range missingCats {
		cats = append(cats, catCount{c, n})
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].n > cats[j].n })
	for _, cc := range cats {
		rec.TypicalMissingEvidence = append(rec.TypicalMissingEvidence, cc.c)
	}
	for _, q := range questionSet {
		rec.RecommendedQuestions = append(rec.RecommendedQuestions, q)
	}
	return rec
}

func humanHypothesis(id string) string {
	return strings.TrimPrefix(id, "hypothesis-")
}
