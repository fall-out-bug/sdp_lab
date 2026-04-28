//go:build sdp_experimental

package main

import (
	"fmt"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
)

// profileToBenchResult converts a CapabilityProfile to a BenchResult for the given task and language.
// If the profile has no capability data for the task:lang key, zero-value duration and test counts are used.
func profileToBenchResult(p *dispatch.CapabilityProfile, taskType, lang string) dispatch.BenchResult {
	key := fmt.Sprintf("%s:%s", taskType, lang)
	cap, hasCap := p.Capabilities[key]

	var dur time.Duration
	var testsPassed, testsTotal int
	if hasCap {
		dur = time.Duration(cap.AvgDuration * float64(time.Minute))
		testsTotal = 10
		testsPassed = int(cap.TestPassRate * float64(testsTotal))
	}

	return dispatch.BenchResult{
		Harness:     p.Harness,
		Provider:    p.Provider,
		Model:       p.Model,
		Task:        taskType,
		TaskType:    taskType,
		Language:    lang,
		Duration:    dur,
		TestsTotal:  testsTotal,
		TestsPassed: testsPassed,
		Timestamp:   time.Now().UTC(),
	}
}
