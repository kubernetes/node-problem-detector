/*
Copyright 2026 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugin

import (
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"k8s.io/utils/clock"
	testclock "k8s.io/utils/clock/testing"

	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
)

const schedulerTestTimeout = 2 * time.Second

type recordingClock struct {
	*testclock.FakeClock
	mu      sync.Mutex
	tickers []clock.Ticker
}

func newRecordingClock() *recordingClock {
	return &recordingClock{FakeClock: testclock.NewFakeClock(time.Unix(0, 0))}
}

func (c *recordingClock) NewTicker(interval time.Duration) clock.Ticker {
	ticker := c.FakeClock.NewTicker(interval)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tickers = append(c.tickers, ticker)
	return ticker
}

func (c *recordingClock) tickerCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.tickers)
}

func (c *recordingClock) ticker(index int) clock.Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tickers[index]
}

type invocation struct {
	rule  string
	count int
}

type invocationKey struct {
	rule  string
	count int
}

type executionRecorder struct {
	mu          sync.Mutex
	counts      map[string]int
	active      map[string]int
	maxActive   map[string]int
	activeTotal int
	highWater   int
	blockers    map[invocationKey]<-chan struct{}
	started     chan invocation
	beforeRun   func(string, int)
}

func newExecutionRecorder() *executionRecorder {
	return &executionRecorder{
		counts:    make(map[string]int),
		active:    make(map[string]int),
		maxActive: make(map[string]int),
		blockers:  make(map[invocationKey]<-chan struct{}),
		started:   make(chan invocation, 100),
	}
}

func (r *executionRecorder) block(rule string, count int, release <-chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.blockers[invocationKey{rule: rule, count: count}] = release
}

func (r *executionRecorder) run(rule cpmtypes.CustomRule) (cpmtypes.Status, string) {
	r.mu.Lock()
	r.counts[rule.Path]++
	count := r.counts[rule.Path]
	r.active[rule.Path]++
	if r.active[rule.Path] > r.maxActive[rule.Path] {
		r.maxActive[rule.Path] = r.active[rule.Path]
	}
	r.activeTotal++
	if r.activeTotal > r.highWater {
		r.highWater = r.activeTotal
	}
	blocker := r.blockers[invocationKey{rule: rule.Path, count: count}]
	beforeRun := r.beforeRun
	r.mu.Unlock()

	if beforeRun != nil {
		beforeRun(rule.Path, count)
	}
	r.started <- invocation{rule: rule.Path, count: count}
	if blocker != nil {
		<-blocker
	}

	r.mu.Lock()
	r.active[rule.Path]--
	r.activeTotal--
	r.mu.Unlock()
	return cpmtypes.OK, rule.Path
}

type executionSnapshot struct {
	counts      map[string]int
	maxActive   map[string]int
	activeTotal int
	highWater   int
}

func (r *executionRecorder) snapshot() executionSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	counts := make(map[string]int, len(r.counts))
	for rule, count := range r.counts {
		counts[rule] = count
	}
	maxActive := make(map[string]int, len(r.maxActive))
	for rule, count := range r.maxActive {
		maxActive[rule] = count
	}
	return executionSnapshot{
		counts:      counts,
		maxActive:   maxActive,
		activeTotal: r.activeTotal,
		highWater:   r.highWater,
	}
}

func schedulerRule(name string, interval *time.Duration) *cpmtypes.CustomRule {
	return &cpmtypes.CustomRule{Path: name, InvokeInterval: interval}
}

func newSchedulerPlugin(t *testing.T, rules []*cpmtypes.CustomRule, globalInterval time.Duration, concurrency int) (*Plugin, *recordingClock, *executionRecorder) {
	t.Helper()
	globalIntervalString := globalInterval.String()
	config := cpmtypes.CustomPluginConfig{Rules: rules}
	config.PluginGlobalConfig.InvokeIntervalString = &globalIntervalString
	config.PluginGlobalConfig.Concurrency = &concurrency
	if err := config.ApplyConfiguration(); err != nil {
		t.Fatalf("ApplyConfiguration() failed: %v", err)
	}
	p := NewPlugin(config)
	fakeClock := newRecordingClock()
	recorder := newExecutionRecorder()
	p.clock = fakeClock
	p.runFunc = recorder.run
	return p, fakeClock, recorder
}

func startPlugin(t *testing.T, p *Plugin, fakeClock *recordingClock, tickerCount int) {
	t.Helper()
	go p.Run()
	waitFor(t, "tickers to be armed", func() bool {
		return fakeClock.HasWaiters() == (tickerCount > 0) && fakeClock.tickerCount() == tickerCount
	})
}

func stopPlugin(t *testing.T, p *Plugin) {
	t.Helper()
	stopped := make(chan struct{})
	go func() {
		p.Stop()
		close(stopped)
	}()
	waitChannel(t, "plugin to stop", stopped)
}

func waitFor(t *testing.T, description string, condition func() bool) {
	t.Helper()
	timer := time.NewTimer(schedulerTestTimeout)
	defer timer.Stop()
	for !condition() {
		select {
		case <-timer.C:
			t.Fatalf("Timed out waiting for %s", description)
		default:
			runtime.Gosched()
		}
	}
}

func waitChannel(t *testing.T, description string, channel <-chan struct{}) {
	t.Helper()
	select {
	case <-channel:
	case <-time.After(schedulerTestTimeout):
		t.Fatalf("Timed out waiting for %s", description)
	}
}

func waitInvocations(t *testing.T, recorder *executionRecorder, count int) []invocation {
	t.Helper()
	invocations := make([]invocation, 0, count)
	for len(invocations) < count {
		select {
		case invocation := <-recorder.started:
			invocations = append(invocations, invocation)
		case <-time.After(schedulerTestTimeout):
			t.Fatalf("Timed out after %d of %d invocation starts", len(invocations), count)
		}
	}
	return invocations
}

func waitResults(t *testing.T, p *Plugin, count int) []cpmtypes.Result {
	t.Helper()
	results := make([]cpmtypes.Result, 0, count)
	for len(results) < count {
		select {
		case result, ok := <-p.resultChan:
			if !ok {
				t.Fatalf("Result channel closed after %d of %d results", len(results), count)
			}
			results = append(results, result)
		case <-time.After(schedulerTestTimeout):
			t.Fatalf("Timed out after %d of %d results", len(results), count)
		}
	}
	return results
}

func stepClock(t *testing.T, fakeClock *recordingClock, duration time.Duration) {
	t.Helper()
	if !fakeClock.HasWaiters() {
		t.Fatal("Fake clock has no armed ticker")
	}
	fakeClock.Step(duration)
}

func assertCounts(t *testing.T, recorder *executionRecorder, wanted map[string]int) {
	t.Helper()
	got := recorder.snapshot().counts
	if !reflect.DeepEqual(got, wanted) {
		t.Fatalf("Invocation counts differ: got %v, wanted %v", got, wanted)
	}
}

func TestPluginSchedulerBootRunsOneCombinedBatch(t *testing.T) {
	interval5 := 5 * time.Second
	interval7 := 7 * time.Second
	interval11 := 11 * time.Second
	rules := []*cpmtypes.CustomRule{
		schedulerRule("five", &interval5),
		schedulerRule("seven", &interval7),
		schedulerRule("eleven", &interval11),
	}
	p, fakeClock, recorder := newSchedulerPlugin(t, rules, 30*time.Second, 3)
	release := make(chan struct{})
	for _, rule := range rules {
		recorder.block(rule.Path, 1, release)
	}
	recorder.beforeRun = func(_ string, _ int) {
		if fakeClock.Waiters() != 3 {
			t.Errorf("Boot execution started with %d tickers; wanted 3", fakeClock.Waiters())
		}
	}

	startPlugin(t, p, fakeClock, 3)
	waitInvocations(t, recorder, 3)
	assertCounts(t, recorder, map[string]int{"five": 1, "seven": 1, "eleven": 1})
	close(release)
	waitResults(t, p, 3)
	stopPlugin(t, p)
}

func TestPluginSchedulerDefaultParityAndSameGroupCoupling(t *testing.T) {
	rules := []*cpmtypes.CustomRule{
		schedulerRule("one", nil),
		schedulerRule("two", nil),
		schedulerRule("three", nil),
	}
	p, fakeClock, recorder := newSchedulerPlugin(t, rules, 10*time.Second, 3)
	release := make(chan struct{})
	recorder.block("one", 2, release)

	startPlugin(t, p, fakeClock, 1)
	waitInvocations(t, recorder, 3)
	waitResults(t, p, 3)
	assertCounts(t, recorder, map[string]int{"one": 1, "two": 1, "three": 1})

	stepClock(t, fakeClock, 10*time.Second)
	waitInvocations(t, recorder, 3)
	waitResults(t, p, 2)
	assertCounts(t, recorder, map[string]int{"one": 2, "two": 2, "three": 2})

	stepClock(t, fakeClock, 10*time.Second)
	if len(fakeClock.ticker(0).C()) != 1 {
		t.Fatalf("Pending same-group tick count is %d; wanted 1", len(fakeClock.ticker(0).C()))
	}
	assertCounts(t, recorder, map[string]int{"one": 2, "two": 2, "three": 2})

	close(release)
	waitResults(t, p, 1)
	waitInvocations(t, recorder, 3)
	waitResults(t, p, 3)
	assertCounts(t, recorder, map[string]int{"one": 3, "two": 3, "three": 3})

	waitFor(t, "the parity group to consume its pending tick", func() bool {
		return len(fakeClock.ticker(0).C()) == 0
	})
	stepClock(t, fakeClock, 10*time.Second)
	waitInvocations(t, recorder, 3)
	waitResults(t, p, 3)
	assertCounts(t, recorder, map[string]int{"one": 4, "two": 4, "three": 4})
	stopPlugin(t, p)
}

func TestPluginSchedulerMixedCadences(t *testing.T) {
	interval7 := 7 * time.Second
	rules := []*cpmtypes.CustomRule{
		schedulerRule("short", &interval7),
		schedulerRule("global", nil),
	}
	p, fakeClock, recorder := newSchedulerPlugin(t, rules, 30*time.Second, 2)
	startPlugin(t, p, fakeClock, 2)
	waitInvocations(t, recorder, 2)
	waitResults(t, p, 2)

	steps := []struct {
		advance       time.Duration
		newExecutions int
		shortCount    int
		globalCount   int
	}{
		{7 * time.Second, 1, 2, 1},
		{7 * time.Second, 1, 3, 1},
		{7 * time.Second, 1, 4, 1},
		{7 * time.Second, 1, 5, 1},
		{2 * time.Second, 1, 5, 2},
		{5 * time.Second, 1, 6, 2},
		{7 * time.Second, 1, 7, 2},
		{7 * time.Second, 1, 8, 2},
		{7 * time.Second, 1, 9, 2},
		{4 * time.Second, 1, 9, 3},
		{3 * time.Second, 1, 10, 3},
		{7 * time.Second, 1, 11, 3},
		{7 * time.Second, 1, 12, 3},
		{7 * time.Second, 1, 13, 3},
		{6 * time.Second, 1, 13, 4},
	}
	for _, step := range steps {
		stepClock(t, fakeClock, step.advance)
		waitInvocations(t, recorder, step.newExecutions)
		waitResults(t, p, step.newExecutions)
		assertCounts(t, recorder, map[string]int{"short": step.shortCount, "global": step.globalCount})
	}
	stopPlugin(t, p)
}

func TestPluginSchedulerEqualParsedIntervalsShareGroup(t *testing.T) {
	explicitIntervalString := "30000ms"
	rules := []*cpmtypes.CustomRule{
		{Path: "unset"},
		{Path: "explicit", InvokeIntervalString: &explicitIntervalString},
	}
	p, fakeClock, recorder := newSchedulerPlugin(t, rules, 30*time.Second, 2)
	groups := p.intervalGroups()
	if len(groups) != 1 || len(groups[0].rules) != 2 {
		t.Fatalf("Parsed-equal rules formed groups %+v; wanted one two-rule group", groups)
	}

	startPlugin(t, p, fakeClock, 1)
	waitInvocations(t, recorder, 2)
	waitResults(t, p, 2)
	stepClock(t, fakeClock, 30*time.Second)
	waitInvocations(t, recorder, 2)
	waitResults(t, p, 2)
	assertCounts(t, recorder, map[string]int{"unset": 2, "explicit": 2})
	stopPlugin(t, p)
}

func TestPluginSchedulerCrossGroupIndependence(t *testing.T) {
	interval5 := 5 * time.Second
	interval7 := 7 * time.Second
	p, fakeClock, recorder := newSchedulerPlugin(t, []*cpmtypes.CustomRule{
		schedulerRule("blocked", &interval5),
		schedulerRule("independent", &interval7),
	}, 30*time.Second, 2)
	release := make(chan struct{})
	recorder.block("blocked", 2, release)

	startPlugin(t, p, fakeClock, 2)
	waitInvocations(t, recorder, 2)
	waitResults(t, p, 2)
	stepClock(t, fakeClock, 5*time.Second)
	waitInvocations(t, recorder, 1)
	stepClock(t, fakeClock, 2*time.Second)
	invocations := waitInvocations(t, recorder, 1)
	if invocations[0] != (invocation{rule: "independent", count: 2}) {
		t.Fatalf("Independent invocation is %+v", invocations[0])
	}
	waitResults(t, p, 1)
	assertCounts(t, recorder, map[string]int{"blocked": 2, "independent": 2})
	close(release)
	waitResults(t, p, 1)
	stopPlugin(t, p)
}

func TestPluginSchedulerConcurrencyReachesLimit(t *testing.T) {
	rules := []*cpmtypes.CustomRule{
		schedulerRule("one", nil),
		schedulerRule("two", nil),
		schedulerRule("three", nil),
		schedulerRule("four", nil),
	}
	p, fakeClock, recorder := newSchedulerPlugin(t, rules, 10*time.Second, 2)
	startPlugin(t, p, fakeClock, 1)
	waitInvocations(t, recorder, 4)
	waitResults(t, p, 4)

	release := make(chan struct{})
	for _, rule := range rules {
		recorder.block(rule.Path, 2, release)
	}
	stepClock(t, fakeClock, 10*time.Second)
	waitInvocations(t, recorder, 2)
	snapshot := recorder.snapshot()
	if snapshot.activeTotal != 2 || snapshot.highWater != 2 {
		t.Fatalf("Concurrency state is active=%d high-water=%d; wanted 2 and 2", snapshot.activeTotal, snapshot.highWater)
	}
	close(release)
	waitInvocations(t, recorder, 2)
	waitResults(t, p, 4)
	if highWater := recorder.snapshot().highWater; highWater != 2 {
		t.Fatalf("Concurrency high-water is %d; wanted 2", highWater)
	}
	stopPlugin(t, p)
}

func TestPluginSchedulerRuleNeverOverlapsAndOverrunCatchesUpOnce(t *testing.T) {
	interval := 5 * time.Second
	p, fakeClock, recorder := newSchedulerPlugin(t, []*cpmtypes.CustomRule{
		schedulerRule("rule", &interval),
	}, 30*time.Second, 2)
	release := make(chan struct{})
	recorder.block("rule", 2, release)

	startPlugin(t, p, fakeClock, 1)
	waitInvocations(t, recorder, 1)
	waitResults(t, p, 1)
	stepClock(t, fakeClock, interval)
	waitInvocations(t, recorder, 1)
	for i := 0; i < 3; i++ {
		stepClock(t, fakeClock, interval)
	}
	assertCounts(t, recorder, map[string]int{"rule": 2})
	close(release)
	waitResults(t, p, 1)
	waitInvocations(t, recorder, 1)
	waitResults(t, p, 1)
	assertCounts(t, recorder, map[string]int{"rule": 3})
	maxActive := recorder.snapshot().maxActive
	if maxActive["rule"] != 1 {
		t.Fatalf("Rule concurrency high-water is %d; wanted 1", maxActive["rule"])
	}
	stopPlugin(t, p)
}

func TestPluginSchedulerConcurrencyOneDoesNotStarveGroups(t *testing.T) {
	interval7 := 7 * time.Second
	p, fakeClock, recorder := newSchedulerPlugin(t, []*cpmtypes.CustomRule{
		schedulerRule("short", &interval7),
		schedulerRule("long", nil),
	}, 30*time.Second, 1)
	startPlugin(t, p, fakeClock, 2)
	waitInvocations(t, recorder, 2)
	waitResults(t, p, 2)

	elapsed := time.Duration(0)
	nextShort := interval7
	nextLong := 30 * time.Second
	shortCount := 1
	longCount := 1
	for elapsed < 210*time.Second {
		nextBoundary := nextShort
		if nextLong < nextBoundary {
			nextBoundary = nextLong
		}
		due := 0
		if nextShort == nextBoundary {
			shortCount++
			nextShort += interval7
			due++
		}
		if nextLong == nextBoundary {
			longCount++
			nextLong += 30 * time.Second
			due++
		}
		stepClock(t, fakeClock, nextBoundary-elapsed)
		elapsed = nextBoundary
		waitInvocations(t, recorder, due)
		waitResults(t, p, due)
		assertCounts(t, recorder, map[string]int{"short": shortCount, "long": longCount})
	}
	assertCounts(t, recorder, map[string]int{"short": 31, "long": 8})
	stopPlugin(t, p)
}

func TestPluginSchedulerZeroRulesWaitsForStop(t *testing.T) {
	p, fakeClock, recorder := newSchedulerPlugin(t, nil, 30*time.Second, 1)
	started := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		close(started)
		p.Run()
		close(finished)
	}()
	<-started
	for i := 0; i < 100; i++ {
		runtime.Gosched()
	}
	if len(recorder.started) != 0 || fakeClock.HasWaiters() {
		t.Fatalf("Zero-rule scheduler started executions or tickers")
	}
	select {
	case <-finished:
		t.Fatal("Zero-rule scheduler returned before stop")
	default:
	}
	stopPlugin(t, p)
	waitChannel(t, "zero-rule Run to return", finished)
}

func TestPluginSchedulerStopWhileIdle(t *testing.T) {
	p, fakeClock, recorder := newSchedulerPlugin(t, []*cpmtypes.CustomRule{
		schedulerRule("idle", nil),
	}, 30*time.Second, 1)
	startPlugin(t, p, fakeClock, 1)
	waitInvocations(t, recorder, 1)
	waitResults(t, p, 1)
	stopPlugin(t, p)
	if _, ok := <-p.resultChan; ok {
		t.Fatal("Result channel remained open after idle stop")
	}
}

func TestPluginSchedulerStopWhileSemaphoreAcquireBlocked(t *testing.T) {
	interval5 := 5 * time.Second
	interval7 := 7 * time.Second
	p, fakeClock, recorder := newSchedulerPlugin(t, []*cpmtypes.CustomRule{
		schedulerRule("holder", &interval5),
		schedulerRule("waiter", &interval7),
	}, 30*time.Second, 1)
	release := make(chan struct{})
	recorder.block("holder", 2, release)
	startPlugin(t, p, fakeClock, 2)
	waitInvocations(t, recorder, 2)
	waitResults(t, p, 2)

	stepClock(t, fakeClock, 5*time.Second)
	waitInvocations(t, recorder, 1)
	stepClock(t, fakeClock, 2*time.Second)
	waitFor(t, "waiter group to consume its tick", func() bool {
		return len(fakeClock.ticker(1).C()) == 0
	})

	stopped := make(chan struct{})
	go func() {
		p.Stop()
		close(stopped)
	}()
	waitFor(t, "stop signal while semaphore acquire is blocked", func() bool {
		select {
		case <-p.tomb.Stopping():
			return true
		default:
			return false
		}
	})
	close(release)
	waitChannel(t, "stop with blocked semaphore acquire", stopped)
	assertCounts(t, recorder, map[string]int{"holder": 2, "waiter": 1})
}

func TestPluginSchedulerStopWithExecutionInFlight(t *testing.T) {
	p, fakeClock, recorder := newSchedulerPlugin(t, []*cpmtypes.CustomRule{
		schedulerRule("in-flight", nil),
	}, 5*time.Second, 1)
	release := make(chan struct{})
	recorder.block("in-flight", 1, release)
	startPlugin(t, p, fakeClock, 1)
	waitInvocations(t, recorder, 1)

	stopped := make(chan struct{})
	go func() {
		p.Stop()
		close(stopped)
	}()
	waitFor(t, "stop signal with execution in flight", func() bool {
		select {
		case <-p.tomb.Stopping():
			return true
		default:
			return false
		}
	})
	select {
	case <-stopped:
		t.Fatal("Stop returned before the in-flight execution finished")
	default:
	}
	close(release)
	waitChannel(t, "stop with execution in flight", stopped)

	stepClock(t, fakeClock, 5*time.Second)
	assertCounts(t, recorder, map[string]int{"in-flight": 1})
	for {
		select {
		case _, ok := <-p.resultChan:
			if !ok {
				if _, open := <-p.resultChan; open {
					t.Fatal("Result channel reopened after close")
				}
				return
			}
		case <-time.After(schedulerTestTimeout):
			t.Fatal("Result channel did not close after in-flight stop")
		}
	}
}
