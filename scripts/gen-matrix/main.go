// gen-matrix reads go test -json output files and emits a markdown compatibility table.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// testEvent is a subset of the go test -json output format.
type testEvent struct {
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
}

const (
	statusPass    = "pass"
	statusFail    = "fail"
	statusSkip    = "skip"
	statusNone    = "none"
	symbolPass    = "ok"
	symbolFail    = "FAIL"
	symbolSkip    = "-"
	symbolUnknown = "?"
)

// rank returns a numeric priority: fail > pass > skip > none.
func rank(s string) int {
	switch s {
	case statusFail:
		return 3
	case statusPass:
		return 2
	case statusSkip:
		return 1
	default:
		return 0
	}
}

func symbol(s string) string {
	switch s {
	case statusPass:
		return symbolPass
	case statusFail:
		return symbolFail
	case statusSkip:
		return symbolSkip
	default:
		return symbolUnknown
	}
}

// featureFromPackage extracts the last path segment of a Go package path.
func featureFromPackage(pkg string) string {
	parts := strings.Split(pkg, "/")
	return parts[len(parts)-1]
}

// targetFromFile derives the target name from a results file name.
// Expected pattern: results-<target>.json
func targetFromFile(path string) string {
	base := filepath.Base(path)
	base = strings.TrimPrefix(base, "results-")
	base = strings.TrimSuffix(base, ".json")
	return base
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gen-matrix results-*.json")
		os.Exit(1)
	}

	// result[target][feature] = status string
	result := make(map[string]map[string]string)
	targets := make([]string, 0)
	features := make(map[string]struct{})

	for _, path := range os.Args[1:] {
		target := targetFromFile(path)
		if _, ok := result[target]; !ok {
			result[target] = make(map[string]string)
			targets = append(targets, target)
		}

		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open %s: %v\n", path, err)
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Bytes()
			var ev testEvent
			if err := json.Unmarshal(line, &ev); err != nil {
				continue
			}
			if ev.Action != statusPass && ev.Action != statusFail && ev.Action != statusSkip {
				continue
			}
			// Package-level events have no Test field.
			if ev.Test != "" {
				continue
			}
			feat := featureFromPackage(ev.Package)
			features[feat] = struct{}{}
			cur := result[target][feat]
			if rank(ev.Action) > rank(cur) {
				result[target][feat] = ev.Action
			}
		}
		f.Close()
	}

	sort.Strings(targets)
	featList := make([]string, 0, len(features))
	for k := range features {
		featList = append(featList, k)
	}
	sort.Strings(featList)

	// Print markdown table.
	// Header.
	header := "| Feature |"
	sep := "| --- |"
	for _, t := range targets {
		header += " " + t + " |"
		sep += " :---: |"
	}
	fmt.Println(header)
	fmt.Println(sep)

	for _, feat := range featList {
		row := "| " + feat + " |"
		for _, t := range targets {
			s := result[t][feat]
			if s == "" {
				s = statusNone
			}
			row += " " + symbol(s) + " |"
		}
		fmt.Println(row)
	}
}
