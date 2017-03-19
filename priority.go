package main

import (
	"path/filepath"
	"strconv"
	"strings"
)

type byPriorityPrefix []string

func (s byPriorityPrefix) Len() int {
	return len(s)
}
func (s byPriorityPrefix) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byPriorityPrefix) Less(i, j int) bool {
	priorityI := getPriority(filepath.Base(s[i]))
	priorityJ := getPriority(filepath.Base(s[j]))
	if priorityI == priorityJ {
		// Sort lexographically.
		return strings.Compare(s[i], s[j]) <= 0
	}
	return priorityI < priorityJ
}

const defaultPriority = 50

// getPriority extracts the priority from the filename in the form $PRIORITY-foo.
// If no priority is specified, we return `defaultPriority`.
func getPriority(filename string) int {
	dashIndex := strings.Index(filename, "-")
	if dashIndex == -1 {
		return defaultPriority
	}

	priority, err := strconv.Atoi(filename[:dashIndex])
	if err != nil {
		return defaultPriority
	}

	return priority
}
