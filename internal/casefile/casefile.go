// Package casefile loads and filters snapcase.json files.
package casefile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type Config struct {
	Version     int      `json:"version"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Wait        int      `json:"wait"`
	PostWait    int      `json:"post_wait"`
	WaitDone    bool     `json:"wait_done"`
	DoneTimeout int      `json:"done_timeout"`
	RPCTimeout  int      `json:"rpc_timeout"`
	LogFile     string   `json:"log_file"`
	LogLevel    string   `json:"log_level"`
	DataHome    string   `json:"data_home"`
	ConfigHome  string   `json:"config_home"`
	RTP         Strings  `json:"rtp"`
}

type Strings []string

func (s *Strings) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = []string{single}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		*s = list
		return nil
	}
	return fmt.Errorf("invalid rtp value")
}

type Case struct {
	Name        string
	Title       string
	Kind        string
	Tags        []string
	Dir         string
	Path        string
	Scenario    string
	Golden      string
	Target      string
	Width       int
	Height      int
	Wait        int
	PostWait    int
	WaitDone    bool
	DoneTimeout int
	RPCTimeout  int
	LogFile     string
	LogLevel    string
	DataHome    string
	ConfigHome  string
	RTP         []string
}

func Load(casePath, root string) (Case, error) {
	data, err := os.ReadFile(casePath)
	if err != nil {
		return Case{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Case{}, fmt.Errorf("failed to parse snapcase.json: %w", err)
	}
	if cfg.Version < 1 {
		return Case{}, errors.New("case version is required")
	}

	caseDir := filepath.Dir(casePath)
	name := filepath.Base(caseDir)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return Case{}, errors.New("case dir name is required")
	}
	kindDir := filepath.Base(filepath.Dir(caseDir))
	if kindDir != "regression" && kindDir != "golden" {
		return Case{}, errors.New("case kind must be regression or golden")
	}
	kind := kindDir

	title := cfg.Title
	if title == "" {
		title = name
	}

	scenarioPath := ""
	if kind == "regression" {
		scenarioPath = filepath.Join(caseDir, "scenario.lua")
	}

	dataHome := cfg.DataHome
	if dataHome == "" {
		dataHome = ".nvim-data"
	}
	configHome := cfg.ConfigHome
	if configHome == "" {
		configHome = ".nvim-config"
	}

	rtp := expandRTP(caseDir, root, cfg.RTP)

	return Case{
		Name:        name,
		Title:       title,
		Kind:        kind,
		Tags:        filterTags(cfg.Tags),
		Dir:         caseDir,
		Path:        caseDir,
		Scenario:    scenarioPath,
		Golden:      filepath.Join(caseDir, "golden.lua"),
		Target:      filepath.Join(caseDir, "target.lua"),
		Width:       positiveOrZero(cfg.Width),
		Height:      positiveOrZero(cfg.Height),
		Wait:        positiveOrZero(cfg.Wait),
		PostWait:    positiveOrZero(cfg.PostWait),
		WaitDone:    cfg.WaitDone,
		DoneTimeout: positiveOrZero(cfg.DoneTimeout),
		RPCTimeout:  positiveOrZero(cfg.RPCTimeout),
		LogFile:     optionalPath(caseDir, cfg.LogFile),
		LogLevel:    cfg.LogLevel,
		DataHome:    filepath.Join(caseDir, dataHome),
		ConfigHome:  filepath.Join(caseDir, configHome),
		RTP:         rtp,
	}, nil
}

func Find(root, casesDir string) ([]Case, []error) {
	casesRoot := root
	if casesDir == "" {
		casesDir = "snapcase"
	}
	if !filepath.IsAbs(casesDir) {
		casesRoot = filepath.Join(root, casesDir)
	} else {
		casesRoot = casesDir
	}
	regressionMatches, err := filepath.Glob(filepath.Join(casesRoot, "regression", "*", "snapcase.json"))
	if err != nil {
		return nil, []error{err}
	}
	goldenMatches, err := filepath.Glob(filepath.Join(casesRoot, "golden", "*", "snapcase.json"))
	if err != nil {
		return nil, []error{err}
	}
	matches := append(regressionMatches, goldenMatches...)
	var out []Case
	var errs []error
	for _, path := range matches {
		c, err := Load(path, root)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, errs
}

func Filter(cases []Case, tags []string, names []string) []Case {
	if len(tags) == 0 && len(names) == 0 {
		return cases
	}
	out := make([]Case, 0, len(cases))
	for _, c := range cases {
		if !matchTags(c.Tags, tags) {
			continue
		}
		if len(names) > 0 && !slices.Contains(names, c.Name) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func FilterByKind(cases []Case, kind string) []Case {
	out := make([]Case, 0, len(cases))
	for _, c := range cases {
		if c.Kind == kind {
			out = append(out, c)
		}
	}
	return out
}

func matchTags(caseTags, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	set := map[string]bool{}
	for _, t := range caseTags {
		set[t] = true
	}
	for _, t := range filter {
		if !set[t] {
			return false
		}
	}
	return true
}

func filterTags(tags []string) []string {
	out := []string{}
	for _, t := range tags {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func expandRTP(caseDir, root string, values []string) []string {
	out := []string{}
	for _, v := range values {
		if v == "" {
			continue
		}
		replaced := strings.ReplaceAll(v, "${CASE}", caseDir)
		replaced = strings.ReplaceAll(replaced, "${ROOT}", root)
		if replaced == "" {
			continue
		}
		if filepath.IsAbs(replaced) {
			out = append(out, filepath.Clean(replaced))
		} else {
			out = append(out, filepath.Clean(filepath.Join(caseDir, replaced)))
		}
	}
	return out
}

func positiveOrZero(value int) int {
	if value <= 0 {
		return 0
	}
	return value
}

func optionalPath(base, value string) string {
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(base, value))
}
