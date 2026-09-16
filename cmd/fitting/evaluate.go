package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"

	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/fitting"
)

type evaluationChart struct {
	ID         int      `json:"id"`
	SongID     int      `json:"song_id"`
	Level      float64  `json:"level"`
	Title      string   `json:"title"`
	Difficulty string   `json:"difficulty"`
	Stored     *float64 `json:"stored"`
}

type evaluationRecord struct {
	User   int `json:"user"`
	Chart  int `json:"chart"`
	Score  int `json:"score"`
	Rating int `json:"rating"`
}

type evaluationVote struct {
	Chart int     `json:"chart"`
	Level float64 `json:"level"`
	Count int     `json:"count"`
}

type evaluationSnapshot struct {
	Charts  []evaluationChart  `json:"charts"`
	Records []evaluationRecord `json:"records"`
	Votes   []evaluationVote   `json:"votes"`
}

// cmdEvaluate replays anonymized local data without a database connection.
// Votes are output for validation only; neither skill nor calibration sees them.
func cmdEvaluate(args []string) {
	fs := flag.NewFlagSet("evaluate", flag.ExitOnError)
	input := fs.String("snapshot", "", "Anonymized JSON produced by scripts/fitting_snapshot.py")
	output := fs.String("output", "", "Output JSON with per-chart comparisons (required)")
	_ = fs.Parse(args)
	if *input == "" || *output == "" {
		fs.Usage()
		os.Exit(2)
	}
	if err := evaluateFile(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func evaluateFile(input, output string) error {
	f, err := os.Open(input)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	var snapshot evaluationSnapshot
	if err := json.NewDecoder(f).Decode(&snapshot); err != nil {
		return err
	}
	config.InitDefaults() // reproducible shipped defaults, never load production secrets
	p := configuredParams()
	byUser := make(map[int][]int)
	for _, record := range snapshot.Records {
		byUser[record.User] = append(byUser[record.User], record.Rating)
	}
	for _, ratings := range byUser {
		sort.Sort(sort.Reverse(sort.IntSlice(ratings)))
	}
	sort.Slice(snapshot.Charts, func(i, j int) bool { return snapshot.Charts[i].ID < snapshot.Charts[j].ID })
	results := make(map[string]map[int]fitting.Result)
	for _, topK := range []int{20, 50} {
		p.SkillTopK = topK
		skills := make(map[int]float64)
		for user, ratings := range byUser {
			k := min(topK, len(ratings))
			sum := 0
			for _, rating := range ratings[:k] {
				sum += rating
			}
			skills[user] = float64(sum) / float64(k) / 100
		}
		groups := make(map[int][]fitting.Sample)
		for _, r := range snapshot.Records {
			groups[r.Chart] = append(groups[r.Chart], fitting.Sample{Username: strconv.Itoa(r.User), Score: r.Score, PlayerSkill: skills[r.User], PlayerRecords: len(byUser[r.User])})
		}
		legacy := p
		legacy.CalibrationEnabled = false
		old := make(map[int]fitting.Result)
		for _, c := range snapshot.Charts {
			old[c.ID] = fitting.ComputeFitting(c.Level, groups[c.ID], legacy)
		}
		results[fmt.Sprintf("legacy_b%d", topK)] = old
		if topK != 20 {
			continue
		}
		cal := &fitting.Calibration{}
		for _, c := range snapshot.Charts {
			cal.AddChart(c.ID, c.Level, groups[c.ID], p)
		}
		calibrated, unscaled := make(map[int]fitting.Result), make(map[int]fitting.Result)
		for _, c := range snapshot.Charts {
			samples := cal.Apply(c.ID, c.Level, groups[c.ID])
			calibrated[c.ID] = fitting.ComputeFitting(c.Level, samples, p)
			q := p
			q.CalibrationScale = 1
			unscaled[c.ID] = fitting.ComputeFitting(c.Level, samples, q)
		}
		results["calibrated_b20"], results["calibrated_unscaled_b20"] = calibrated, unscaled
	}
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	return json.NewEncoder(out).Encode(struct {
		Charts  []evaluationChart                 `json:"charts"`
		Votes   []evaluationVote                  `json:"votes"`
		Results map[string]map[int]fitting.Result `json:"results"`
	}{snapshot.Charts, snapshot.Votes, results})
}
