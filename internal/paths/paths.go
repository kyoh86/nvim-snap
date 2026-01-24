// Package paths provides case/result path helpers.
package paths

import "path/filepath"

func ResolveCasesRoot(absRoot, casesDir string) string {
	if casesDir == "" {
		casesDir = "snapcase"
	}
	if filepath.IsAbs(casesDir) {
		return casesDir
	}
	return filepath.Join(absRoot, casesDir)
}

func ResolveResultsRoot(absRoot, casesDir string) string {
	return filepath.Join(ResolveCasesRoot(absRoot, casesDir), ".result")
}
