// Copyright 2025 The Viction Authors
// (modifications)
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"container/heap"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// Value of ethconfig.Defaults.Miner.Recommit; importing ethconfig from this package is an import cycle.
	simDefaultRecommit = 3 * time.Second
	// Sleep of BlockFetcher.retryPosvBlock after a future block's timestamp is reached.
	simFutureBlockMargin = 50 * time.Millisecond
	// Virtual clock origin, far enough from zero for a skewed clock to stay positive.
	simStartTime = time.Hour
	// Number of seeds each scenario is averaged over.
	simSeedCount = 3
	// Seed of the single runs the assertions are made on.
	simTestSeed = 1
	// Lag of a network where blocks spread well within a block period.
	simFastLag = 300 * time.Millisecond
)

var (
	simValidatorCount = int(params.VictionChainConfig.Viction.ValidatorMaxCount)
	simEpoch          = params.VictionChainConfig.Posv.Epoch
	simBlockPeriod    = time.Duration(params.VictionChainConfig.Posv.Period) * time.Second
	// One epoch at full speed.
	simDuration = simBlockPeriod * time.Duration(simEpoch)

	// Candidate values of posvWaitPeriod; 10s is the value the chain has run with.
	simWait10s      = posvWaitPolicy{name: "10s", waitPeriod: 10 * time.Second, waitPeriodCheckpoint: defaultPosvWaitPeriodCheckpoint}
	simWait5s       = posvWaitPolicy{name: "5s", waitPeriod: 5 * time.Second, waitPeriodCheckpoint: defaultPosvWaitPeriodCheckpoint}
	simWait4s       = posvWaitPolicy{name: "4s", waitPeriod: 4 * time.Second, waitPeriodCheckpoint: defaultPosvWaitPeriodCheckpoint}
	simWait3s       = posvWaitPolicy{name: "3s", waitPeriod: 3 * time.Second, waitPeriodCheckpoint: defaultPosvWaitPeriodCheckpoint}
	simWait2s       = posvWaitPolicy{name: "2s", waitPeriod: 2 * time.Second, waitPeriodCheckpoint: defaultPosvWaitPeriodCheckpoint}
	simWaitPolicies = []posvWaitPolicy{simWait10s, simWait5s, simWait4s, simWait3s, simWait2s}
)

// posvWaitPolicy is one candidate pair of out-of-turn wait periods.
type posvWaitPolicy struct {
	name                 string
	waitPeriod           time.Duration
	waitPeriodCheckpoint time.Duration
}

// posvSimScenario describes the network a simulation runs on. Delays are assumptions, not measurements.
type posvSimScenario struct {
	name string
	// Mean delay between a block's release and its import by another node, attestation included.
	// Each of the two hops (creator to attestor, attestor to node) takes 25% to 75% of it.
	meanLag time.Duration
	// Share of blocks whose attestation takes lagSpike longer.
	lagSpikeChance float64
	lagSpike       time.Duration
	// Validators that neither create, attest, nor import blocks.
	offline []int
	// Extra one-way latency of every message to or from a validator.
	slow map[int]time.Duration
	// Each validator's clock is off by a random amount within +/- this.
	maxClockSkew time.Duration
	// miner.recommit of every validator; zero selects simDefaultRecommit.
	recommit time.Duration
	// No transaction is ever pending. Otherwise every validator has some pending at each recommit tick.
	hasEmptyTxPool bool
	// Zero selects the mainnet validator count.
	validatorCount int
	// Time during which validators build; zero selects one epoch at full speed.
	duration time.Duration
}

// Fills the settings a scenario leaves at zero.
func (scenario posvSimScenario) withDefaults() posvSimScenario {
	if scenario.recommit == 0 {
		scenario.recommit = simDefaultRecommit
	}
	if scenario.validatorCount == 0 {
		scenario.validatorCount = simValidatorCount
	}
	if scenario.duration == 0 {
		scenario.duration = simDuration
	}
	return scenario
}

// simDelivery is the delay after which one node imports a released block.
type simDelivery struct {
	node  int
	delay time.Duration
}

// simNetwork is the fake network both the model and the live run deliver blocks through: it picks the attestor of
// a block and the latency of each hop. Node validatorCount is the observer. Not safe for concurrent use.
type simNetwork struct {
	scenario      posvSimScenario
	random        *rand.Rand
	attestorOrder []int
	isOffline     map[int]bool
}

func newSimNetwork(scenario posvSimScenario, random *rand.Rand) *simNetwork {
	network := &simNetwork{
		scenario:      scenario,
		random:        random,
		attestorOrder: random.Perm(scenario.validatorCount),
		isOffline:     map[int]bool{},
	}
	for _, index := range scenario.offline {
		network.isOffline[index] = true
	}
	return network
}

// Mirrors getCreatorAttestorPairs with TIPRandomize active.
func (n *simNetwork) attestor(creator int, number uint64) int {
	validatorCount := uint64(n.scenario.validatorCount)
	offset := ((number % simEpoch) / validatorCount) % validatorCount
	return int((uint64(n.attestorOrder[creator]) + offset) % validatorCount)
}

func (n *simNetwork) hopLatency(from, to int) time.Duration {
	if from == to {
		return 0
	}
	quarter := n.scenario.meanLag / 4
	return quarter + time.Duration(n.random.Int63n(int64(2*quarter)+1)) + n.scenario.slow[from] + n.scenario.slow[to]
}

// Routes a released block to its attestor, then from there to every online node. A block whose attestor is
// offline is never attested and reaches nobody.
func (n *simNetwork) deliveries(creator int, number uint64) (deliveries []simDelivery, isAttested bool) {
	return n.deliveriesVia(creator, n.attestor(creator, number))
}

func (n *simNetwork) deliveriesVia(creator, attestor int) (deliveries []simDelivery, isAttested bool) {
	if n.isOffline[attestor] {
		return nil, false
	}
	attestDelay := n.hopLatency(creator, attestor)
	if n.random.Float64() < n.scenario.lagSpikeChance {
		attestDelay += n.scenario.lagSpike
	}
	for node := 0; node <= n.scenario.validatorCount; node++ {
		if !n.isOffline[node] {
			deliveries = append(deliveries, simDelivery{node: node, delay: attestDelay + n.hopLatency(attestor, node)})
		}
	}
	return deliveries, true
}

type simBlock struct {
	parent    *simBlock
	number    uint64
	creator   int
	timestamp int64 // unix seconds, as header.Time
	// Circular distance from the parent's creator; 1 is in-turn.
	distance        int
	totalDifficulty int
	releasedAt      time.Duration
}

type simNode struct {
	index       int
	isOnline    bool
	isMining    bool
	clockOffset time.Duration

	head           *simBlock
	imported       map[*simBlock]struct{}
	awaitingParent map[*simBlock][]*simBlock
	// Mirrors worker.posvLastParentCommitHash.
	committedParent *simBlock
	sealGeneration  uint64
	tickGeneration  uint64

	reorgCount   int
	deepestReorg int
}

type simEventKind int

const (
	simDeliver simEventKind = iota
	simRelease
	simTick
)

// simCommitTrigger is what made a validator reconsider building, as in worker.newWorkLoop.
type simCommitTrigger int

const (
	simNewHead simCommitTrigger = iota
	simRecommitTick
)

type simEvent struct {
	at         time.Duration
	sequence   uint64
	kind       simEventKind
	node       *simNode
	block      *simBlock
	generation uint64
}

// simEventQueue orders events by time, then by scheduling order, so that a run is reproducible.
type simEventQueue []simEvent

func (q simEventQueue) Len() int { return len(q) }
func (q simEventQueue) Less(i, j int) bool {
	if q[i].at != q[j].at {
		return q[i].at < q[j].at
	}
	return q[i].sequence < q[j].sequence
}
func (q simEventQueue) Swap(i, j int)       { q[i], q[j] = q[j], q[i] }
func (q *simEventQueue) Push(x interface{}) { *q = append(*q, x.(simEvent)) }
func (q *simEventQueue) Pop() interface{} {
	old := *q
	last := old[len(old)-1]
	*q = old[:len(old)-1]
	return last
}

// posvSim replays PoSV block production on a virtual clock. It models these parts of the node:
//   - worker.newWorkLoop: a validator reconsiders building on every new head and every recommit tick.
//   - worker.commitNewWork, worker.preCommitNewWorkForPosv: one block per parent, out-of-turn only after the wait,
//     and on a recommit tick only when transactions are pending.
//   - Posv.Prepare, Posv.Seal: whole-second timestamps, release at the timestamp, no two blocks in a row.
//   - worker.taskLoop: a block built on a new head after its slot has passed leaves together with its pre-sealed
//     empty twin.
//   - BlockFetcher.importBlocks: a block spreads only after its creator's attestor signed it.
//   - BlockChain.writeBlockWithState: heaviest chain wins, a tie keeps the local block or flips a coin.
//
// The validator set is fixed for the run, transactions and block execution cost are not modelled, and the
// last node is a full node that only observes.
type posvSim struct {
	scenario posvSimScenario
	policy   posvWaitPolicy
	random   *rand.Rand

	now      time.Duration
	deadline time.Duration
	queue    simEventQueue
	sequence uint64

	nodes    []*simNode
	network  *simNetwork
	start    *simBlock
	attested []*simBlock
}

func newPosvSim(scenario posvSimScenario, policy posvWaitPolicy, seed int64) *posvSim {
	scenario = scenario.withDefaults()
	sim := &posvSim{
		scenario: scenario,
		policy:   policy,
		random:   rand.New(rand.NewSource(seed)),
		now:      simStartTime,
		deadline: simStartTime + scenario.duration,
	}
	sim.network = newSimNetwork(scenario, sim.random)
	// Starts right after a checkpoint, on a block of the last validator, so validator 0 is in-turn.
	sim.start = &simBlock{
		number:     1,
		creator:    scenario.validatorCount - 1,
		timestamp:  int64(simStartTime / time.Second),
		distance:   1,
		releasedAt: simStartTime,
	}
	for index := 0; index <= scenario.validatorCount; index++ {
		isValidator := index < scenario.validatorCount
		isOnline := !sim.network.isOffline[index]
		node := &simNode{
			index:          index,
			isOnline:       isOnline,
			isMining:       isValidator && isOnline,
			head:           sim.start,
			imported:       map[*simBlock]struct{}{sim.start: {}},
			awaitingParent: map[*simBlock][]*simBlock{},
		}
		if isValidator && scenario.maxClockSkew > 0 {
			node.clockOffset = time.Duration(sim.random.Int63n(int64(2*scenario.maxClockSkew)+1)) - scenario.maxClockSkew
		}
		sim.nodes = append(sim.nodes, node)
	}
	return sim
}

// Runs the simulation until no validator builds anymore and every broadcast block has been delivered.
func (s *posvSim) run() {
	for _, node := range s.nodes {
		s.commitNewWork(node, simNewHead)
		s.resetTick(node)
	}
	for s.queue.Len() > 0 {
		event := heap.Pop(&s.queue).(simEvent)
		s.now = event.at
		switch event.kind {
		case simDeliver:
			s.importBlock(event.node, event.block)
		case simRelease:
			if event.generation == event.node.sealGeneration {
				s.broadcast(event.block)
			}
		case simTick:
			if event.generation == event.node.tickGeneration {
				s.commitNewWork(event.node, simRecommitTick)
				s.resetTick(event.node)
			}
		}
	}
}

func (s *posvSim) schedule(event simEvent) {
	s.sequence++
	event.sequence = s.sequence
	heap.Push(&s.queue, event)
}

func (s *posvSim) resetTick(node *simNode) {
	node.tickGeneration++
	if !node.isMining || s.now >= s.deadline {
		return
	}
	s.schedule(simEvent{at: s.now + s.scenario.recommit, kind: simTick, node: node, generation: node.tickGeneration})
}

func (s *posvSim) localTime(node *simNode) time.Duration {
	return s.now + node.clockOffset
}

// Mirrors the turn and wait rule of worker.preCommitNewWorkForPosv.
func (s *posvSim) mayBuildOn(node *simNode, parent *simBlock) bool {
	if (parent.creator+1)%s.scenario.validatorCount == node.index {
		return true
	}
	hops := common.CircularDistance(node.index, parent.creator, s.scenario.validatorCount) - 1
	gap := s.policy.waitPeriod * time.Duration(hops)
	if nearest := simEpoch - parent.number%simEpoch; uint64(hops) >= nearest {
		gap = s.policy.waitPeriodCheckpoint * time.Duration(hops)
	}
	waited := s.localTime(node) - time.Duration(parent.timestamp)*time.Second
	if waited < 0 {
		waited = 0
	}
	return gap <= waited
}

// Mirrors worker.commitNewWork, Posv.Prepare and Posv.Seal for one validator.
func (s *posvSim) commitNewWork(node *simNode, trigger simCommitTrigger) {
	if !node.isMining || s.now >= s.deadline {
		return
	}
	parent := node.head
	if node.committedParent == parent || !s.mayBuildOn(node, parent) {
		return
	}
	// Only a new head pre-seals an empty block; a recommit tick wants transactions to fill one with.
	if trigger == simRecommitTick && s.scenario.hasEmptyTxPool {
		return
	}
	node.committedParent = parent
	// taskLoop interrupts the previous sealing before calling Seal, also when Seal then refuses.
	node.sealGeneration++
	number := parent.number + 1
	if parent.creator == node.index && number%simEpoch != 0 {
		return
	}
	timestamp := parent.timestamp + int64(simBlockPeriod/time.Second)
	if localUnix := int64(s.localTime(node) / time.Second); timestamp < localUnix {
		timestamp = localUnix
	}
	distance := common.CircularDistance(node.index, parent.creator, s.scenario.validatorCount)
	block := &simBlock{
		parent:          parent,
		number:          number,
		creator:         node.index,
		timestamp:       timestamp,
		distance:        distance,
		totalDifficulty: parent.totalDifficulty + s.scenario.validatorCount - distance + 1,
	}
	releaseAt := time.Duration(timestamp)*time.Second - node.clockOffset
	if releaseAt > s.now {
		s.schedule(simEvent{at: releaseAt, kind: simRelease, node: node, block: block, generation: node.sealGeneration})
		return
	}
	// A new head pre-seals an empty block before the full one. Sealing the full block cancels it, unless the
	// slot has already passed and it left at once: then both are released.
	if trigger == simNewHead && !s.scenario.hasEmptyTxPool {
		emptyBlock := *block
		s.schedule(simEvent{at: s.now, kind: simRelease, node: node, block: &emptyBlock, generation: node.sealGeneration})
	}
	s.schedule(simEvent{at: s.now, kind: simRelease, node: node, block: block, generation: node.sealGeneration})
}

func (s *posvSim) broadcast(block *simBlock) {
	block.releasedAt = s.now
	deliveries, isAttested := s.network.deliveries(block.creator, block.number)
	if !isAttested {
		return
	}
	s.attested = append(s.attested, block)
	for _, delivery := range deliveries {
		s.schedule(simEvent{at: s.now + delivery.delay, kind: simDeliver, node: s.nodes[delivery.node], block: block})
	}
}

func (s *posvSim) importBlock(node *simNode, block *simBlock) {
	if _, isKnown := node.imported[block]; isKnown {
		return
	}
	if _, hasParent := node.imported[block.parent]; !hasParent {
		node.awaitingParent[block.parent] = append(node.awaitingParent[block.parent], block)
		return
	}
	// Fetcher holds a block from the future until the local clock reaches its timestamp.
	if readyAt := time.Duration(block.timestamp)*time.Second - node.clockOffset; readyAt > s.now {
		s.schedule(simEvent{at: readyAt + simFutureBlockMargin, kind: simDeliver, node: node, block: block})
		return
	}
	node.imported[block] = struct{}{}
	if s.shouldAdopt(node, block) {
		s.adoptHead(node, block)
	}
	children := node.awaitingParent[block]
	delete(node.awaitingParent, block)
	for _, child := range children {
		s.importBlock(node, child)
	}
}

// Mirrors the fork choice of BlockChain.writeBlockWithState with Ethereum.shouldPreserve.
func (s *posvSim) shouldAdopt(node *simNode, block *simBlock) bool {
	head := node.head
	if block.totalDifficulty != head.totalDifficulty {
		return block.totalDifficulty > head.totalDifficulty
	}
	if block.number != head.number {
		return block.number < head.number
	}
	if head.creator == node.index {
		return false
	}
	return block.creator == node.index || s.random.Float64() < 0.5
}

func (s *posvSim) adoptHead(node *simNode, block *simBlock) {
	if dropped := droppedBlockCount(node.head, block); dropped > 0 {
		node.reorgCount++
		if dropped > node.deepestReorg {
			node.deepestReorg = dropped
		}
	}
	node.head = block
	s.commitNewWork(node, simNewHead)
	s.resetTick(node)
}

// Counts the blocks of oldHead's chain that are not on newHead's chain.
func droppedBlockCount(oldHead, newHead *simBlock) int {
	dropped := 0
	for oldHead.number > newHead.number {
		oldHead = oldHead.parent
		dropped++
	}
	for newHead.number > oldHead.number {
		newHead = newHead.parent
	}
	for oldHead != newHead {
		oldHead, newHead = oldHead.parent, newHead.parent
		dropped++
	}
	return dropped
}

// posvSimResult summarizes one run, with the observer's chain taken as canonical.
type posvSimResult struct {
	canonicalBlocks int
	meanBlockTime   time.Duration
	// Longest wait between the release of two consecutive canonical blocks.
	worstBlockGap time.Duration
	// Heights at which more than one attested block was broadcast.
	forkedHeights int
	// Attested blocks that did not become canonical.
	orphanedBlocks int
	// In-turn blocks on a canonical parent that did not become canonical.
	lostInTurnBlocks int
	// Canonical blocks created out-of-turn.
	backupBlocks int
	// Head switches of the observer that dropped at least one block.
	observerReorgs int
	deepestReorg   int
	// Distinct total difficulties of the heads the online nodes ended on once the network went quiet. Equally
	// heavy heads are a tie the next block settles; more than one weight means a node missed a heavier chain.
	headWeights     int
	blocksByCreator []int
}

// posvRun is what a run, modelled or live, leaves behind to be summarized.
type posvRun struct {
	validatorCount int
	start          *simBlock
	attested       []*simBlock
	observerHead   *simBlock
	observerReorgs int
	deepestReorg   int
	headWeights    int
}

func (run posvRun) summarize() posvSimResult {
	result := posvSimResult{
		observerReorgs:  run.observerReorgs,
		deepestReorg:    run.deepestReorg,
		headWeights:     run.headWeights,
		blocksByCreator: make([]int, run.validatorCount),
	}
	isCanonical := map[*simBlock]bool{run.start: true}
	for block := run.observerHead; block != run.start; block = block.parent {
		isCanonical[block] = true
		result.canonicalBlocks++
		result.blocksByCreator[block.creator]++
		if block.distance != 1 {
			result.backupBlocks++
		}
		if gap := block.releasedAt - block.parent.releasedAt; gap > result.worstBlockGap {
			result.worstBlockGap = gap
		}
	}
	if result.canonicalBlocks > 0 {
		result.meanBlockTime = (run.observerHead.releasedAt - run.start.releasedAt) / time.Duration(result.canonicalBlocks)
	}
	blocksAtHeight := map[uint64]int{}
	for _, block := range run.attested {
		blocksAtHeight[block.number]++
		if isCanonical[block] {
			continue
		}
		result.orphanedBlocks++
		if block.distance == 1 && isCanonical[block.parent] {
			result.lostInTurnBlocks++
		}
	}
	for _, count := range blocksAtHeight {
		if count > 1 {
			result.forkedHeights++
		}
	}
	return result
}

func (s *posvSim) result() posvSimResult {
	observer := s.nodes[s.scenario.validatorCount]
	headWeights := map[int]struct{}{}
	for _, node := range s.nodes {
		if node.isOnline {
			headWeights[node.head.totalDifficulty] = struct{}{}
		}
	}
	return posvRun{
		validatorCount: s.scenario.validatorCount,
		start:          s.start,
		attested:       s.attested,
		observerHead:   observer.head,
		observerReorgs: observer.reorgCount,
		deepestReorg:   observer.deepestReorg,
		headWeights:    len(headWeights),
	}.summarize()
}

func runPosvSim(scenario posvSimScenario, policy posvWaitPolicy, seed int64) posvSimResult {
	sim := newPosvSim(scenario, policy, seed)
	sim.run()
	return sim.result()
}

// posvSimAverage holds per-epoch means over simSeedCount runs, and the worst case for gaps and reorg depth.
type posvSimAverage struct {
	canonicalBlocks  float64
	meanBlockTime    time.Duration
	worstBlockGap    time.Duration
	forkedHeights    float64
	orphanedBlocks   float64
	lostInTurnBlocks float64
	backupBlocks     float64
	observerReorgs   float64
	deepestReorg     int
	mostHeadWeights  int
}

func averagePosvSim(scenario posvSimScenario, policy posvWaitPolicy) posvSimAverage {
	var average posvSimAverage
	for seed := int64(1); seed <= simSeedCount; seed++ {
		result := runPosvSim(scenario, policy, seed)
		average.canonicalBlocks += float64(result.canonicalBlocks) / simSeedCount
		average.meanBlockTime += result.meanBlockTime / simSeedCount
		average.forkedHeights += float64(result.forkedHeights) / simSeedCount
		average.orphanedBlocks += float64(result.orphanedBlocks) / simSeedCount
		average.lostInTurnBlocks += float64(result.lostInTurnBlocks) / simSeedCount
		average.backupBlocks += float64(result.backupBlocks) / simSeedCount
		average.observerReorgs += float64(result.observerReorgs) / simSeedCount
		if result.worstBlockGap > average.worstBlockGap {
			average.worstBlockGap = result.worstBlockGap
		}
		if result.deepestReorg > average.deepestReorg {
			average.deepestReorg = result.deepestReorg
		}
		if result.headWeights > average.mostHeadWeights {
			average.mostHeadWeights = result.headWeights
		}
	}
	return average
}

func everyNthValidator(step, first int) []int {
	var indexes []int
	for index := first; index < simValidatorCount; index += step {
		indexes = append(indexes, index)
	}
	return indexes
}

func posvSimScenarios() []posvSimScenario {
	return []posvSimScenario{
		{name: "healthy, 0.3s lag", meanLag: simFastLag},
		{name: "healthy, 1s lag", meanLag: time.Second},
		{name: "healthy, 1.5s lag", meanLag: 1500 * time.Millisecond},
		{name: "congested, 2.5s lag", meanLag: 2500 * time.Millisecond},
		{name: "5% of blocks 1.5s late", meanLag: simFastLag, lagSpikeChance: 0.05, lagSpike: 1500 * time.Millisecond},
		{name: "5% of blocks 2.5s late", meanLag: simFastLag, lagSpikeChance: 0.05, lagSpike: 2500 * time.Millisecond},
		{name: "1 offline", meanLag: simFastLag, offline: []int{10}},
		{name: "1 offline, 1s lag", meanLag: time.Second, offline: []int{10}},
		{name: "2 consecutive offline", meanLag: simFastLag, offline: []int{10, 11}},
		{name: "3 consecutive offline", meanLag: simFastLag, offline: []int{10, 11, 12}},
		{name: "10% offline, scattered", meanLag: simFastLag, offline: everyNthValidator(10, 5)},
		{name: "1 validator 1s slower", meanLag: simFastLag, slow: map[int]time.Duration{10: time.Second}},
		{name: "1 validator 2s slower", meanLag: simFastLag, slow: map[int]time.Duration{10: 2 * time.Second}},
		{name: "1 validator 3s slower", meanLag: simFastLag, slow: map[int]time.Duration{10: 3 * time.Second}},
		{name: "clocks off by up to 0.25s", meanLag: simFastLag, maxClockSkew: 250 * time.Millisecond},
		{name: "clocks off by up to 1s", meanLag: simFastLag, maxClockSkew: time.Second},
		{name: "1 offline, recommit 1s", meanLag: simFastLag, offline: []int{10}, recommit: time.Second},
		{name: "healthy 1s lag, recommit 1s", meanLag: time.Second, recommit: time.Second},
		{name: "1 offline, empty tx pool", meanLag: simFastLag, offline: []int{10}, hasEmptyTxPool: true},
	}
}

// Prints, for every scenario, how each candidate wait period performs over one epoch. Run with -v to read the table.
func TestPosvWaitPeriodComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("simulates every scenario; skipped in short mode")
	}
	var table strings.Builder
	fmt.Fprintf(&table, "\n%-28s %-5s %7s %9s %9s %7s %8s %12s %7s %7s %6s\n",
		"scenario", "wait", "blocks", "avg time", "worst gap", "forked", "orphaned", "lost in-turn", "backup", "reorgs", "depth")
	for _, scenario := range posvSimScenarios() {
		for _, policy := range simWaitPolicies {
			average := averagePosvSim(scenario, policy)
			fmt.Fprintf(&table, "%-28s %-5s %7.1f %9s %9s %7.1f %8.1f %12.1f %7.1f %7.1f %6d\n",
				scenario.name, policy.name, average.canonicalBlocks,
				average.meanBlockTime.Round(time.Millisecond), average.worstBlockGap.Round(10*time.Millisecond),
				average.forkedHeights, average.orphanedBlocks, average.lostInTurnBlocks, average.backupBlocks,
				average.observerReorgs, average.deepestReorg)
			if average.mostHeadWeights != 1 {
				t.Errorf("%s, wait %s: online nodes ended on heads of %d weights, want 1", scenario.name, policy.name, average.mostHeadWeights)
			}
		}
		table.WriteString("\n")
	}
	t.Log(table.String())
}

func TestPosvSimHealthyNetworkHasNoForks(t *testing.T) {
	scenario := posvSimScenario{meanLag: simFastLag}
	for _, policy := range simWaitPolicies {
		t.Run(policy.name, func(t *testing.T) {
			result := runPosvSim(scenario, policy, simTestSeed)

			if result.orphanedBlocks != 0 || result.backupBlocks != 0 {
				t.Errorf("got %d orphaned and %d out-of-turn blocks, want none", result.orphanedBlocks, result.backupBlocks)
			}
			if result.meanBlockTime != simBlockPeriod {
				t.Errorf("mean block time = %s, want %s", result.meanBlockTime, simBlockPeriod)
			}
		})
	}
}

func TestPosvSimOfflineValidatorSlotIsTakenWithinOneTick(t *testing.T) {
	scenario := posvSimScenario{meanLag: simFastLag, offline: []int{10}}
	worstLag := scenario.meanLag * 3 / 2
	for _, policy := range simWaitPolicies {
		t.Run(policy.name, func(t *testing.T) {
			result := runPosvSim(scenario, policy, simTestSeed)

			if result.worstBlockGap < policy.waitPeriod {
				t.Errorf("worst gap = %s, want at least the wait period %s", result.worstBlockGap, policy.waitPeriod)
			}
			if limit := policy.waitPeriod + simDefaultRecommit + worstLag; result.worstBlockGap >= limit {
				t.Errorf("worst gap = %s, want below %s", result.worstBlockGap, limit)
			}
		})
	}
}

func TestPosvSimWaitsUpToRecommitRecoverOnSameTick(t *testing.T) {
	scenario := posvSimScenario{meanLag: simFastLag, offline: []int{10}}

	threeSeconds := runPosvSim(scenario, simWait3s, simTestSeed)
	twoSeconds := runPosvSim(scenario, simWait2s, simTestSeed)

	if threeSeconds.worstBlockGap != twoSeconds.worstBlockGap {
		t.Errorf("worst gap = %s with 3s, %s with 2s, want equal", threeSeconds.worstBlockGap, twoSeconds.worstBlockGap)
	}
	if threeSeconds.canonicalBlocks != twoSeconds.canonicalBlocks {
		t.Errorf("blocks = %d with 3s, %d with 2s, want equal", threeSeconds.canonicalBlocks, twoSeconds.canonicalBlocks)
	}
	if threeSeconds.orphanedBlocks != 0 || twoSeconds.orphanedBlocks != 0 {
		t.Errorf("orphaned blocks = %d with 3s, %d with 2s, want none", threeSeconds.orphanedBlocks, twoSeconds.orphanedBlocks)
	}
}

func TestPosvSimConsecutiveOfflineValidators(t *testing.T) {
	scenario := posvSimScenario{meanLag: simFastLag, offline: []int{10, 11}}
	tests := []struct {
		name        string
		policy      posvWaitPolicy
		wantOrphans bool
	}{
		{name: "3s wait gives each backup its own tick", policy: simWait3s, wantOrphans: false},
		{name: "2s wait lets two backups build on the same tick", policy: simWait2s, wantOrphans: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runPosvSim(scenario, tt.policy, simTestSeed)

			if hasOrphans := result.orphanedBlocks > 0; hasOrphans != tt.wantOrphans {
				t.Errorf("orphaned blocks = %d, want any = %t", result.orphanedBlocks, tt.wantOrphans)
			}
		})
	}
}

func TestPosvSimRecommitBelowWaitPeriod(t *testing.T) {
	scenario := posvSimScenario{meanLag: simFastLag, recommit: time.Second}
	tests := []struct {
		name            string
		policy          posvWaitPolicy
		wantForkedShare float64
	}{
		{name: "3s wait still outlasts the in-turn block", policy: simWait3s, wantForkedShare: 0},
		{name: "2s wait races the in-turn block", policy: simWait2s, wantForkedShare: 0.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runPosvSim(scenario, tt.policy, simTestSeed)

			forkedShare := float64(result.forkedHeights) / float64(result.canonicalBlocks)
			if tt.wantForkedShare == 0 && result.forkedHeights != 0 {
				t.Errorf("forked heights = %d, want none", result.forkedHeights)
			}
			if forkedShare < tt.wantForkedShare {
				t.Errorf("forked share = %.2f, want at least %.2f", forkedShare, tt.wantForkedShare)
			}
		})
	}
}

func TestPosvSimSlowValidatorBlocks(t *testing.T) {
	const slowValidator = 10
	scenario := posvSimScenario{meanLag: simFastLag, slow: map[int]time.Duration{slowValidator: 3 * time.Second}}
	minBlocks := int(params.VictionChainConfig.Viction.ValidatorMinBlockPerEpochCount)
	tests := []struct {
		name          string
		policy        posvWaitPolicy
		wantPenalized bool
	}{
		{name: "10s wait keeps the blocks of a validator 3s behind each way", policy: simWait10s, wantPenalized: false},
		{name: "3s wait orphans them", policy: simWait3s, wantPenalized: true},
		{name: "2s wait orphans them", policy: simWait2s, wantPenalized: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runPosvSim(scenario, tt.policy, simTestSeed)

			blocks := result.blocksByCreator[slowValidator]
			if isPenalized := blocks < minBlocks; isPenalized != tt.wantPenalized {
				t.Errorf("slow validator has %d canonical blocks, penalized = %t, want %t", blocks, isPenalized, tt.wantPenalized)
			}
		})
	}
}

func TestPosvSimEmptyTxPoolHaltsAtOfflineValidator(t *testing.T) {
	const offlineValidator = 10
	scenario := posvSimScenario{meanLag: simFastLag, offline: []int{offlineValidator}, hasEmptyTxPool: true}
	for _, policy := range simWaitPolicies {
		t.Run(policy.name, func(t *testing.T) {
			result := runPosvSim(scenario, policy, simTestSeed)

			if result.canonicalBlocks > offlineValidator {
				t.Errorf("chain grew by %d blocks, want it stopped at the offline validator's slot after %d", result.canonicalBlocks, offlineValidator)
			}
			if result.backupBlocks != 0 {
				t.Errorf("got %d out-of-turn blocks, want none without pending transactions", result.backupBlocks)
			}
		})
	}
}

func TestPosvSimLateHeadReleasesEmptyTwin(t *testing.T) {
	tests := []struct {
		name           string
		hasEmptyTxPool bool
		wantForks      bool
	}{
		{name: "pending transactions put a full block next to the empty one", hasEmptyTxPool: false, wantForks: true},
		{name: "empty tx pool leaves the empty block alone", hasEmptyTxPool: true, wantForks: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := posvSimScenario{meanLag: 1500 * time.Millisecond, hasEmptyTxPool: tt.hasEmptyTxPool}

			result := runPosvSim(scenario, simWait10s, simTestSeed)

			if hasForks := result.forkedHeights > 0; hasForks != tt.wantForks {
				t.Errorf("forked heights = %d, want any = %t", result.forkedHeights, tt.wantForks)
			}
			if result.backupBlocks != 0 {
				t.Errorf("got %d out-of-turn blocks, want none within a 10s wait", result.backupBlocks)
			}
		})
	}
}

func TestPosvSimIsReproducible(t *testing.T) {
	scenario := posvSimScenario{meanLag: 1500 * time.Millisecond, offline: []int{10}}

	first := runPosvSim(scenario, simWait2s, simTestSeed)
	second := runPosvSim(scenario, simWait2s, simTestSeed)

	if !reflect.DeepEqual(first, second) {
		t.Errorf("two runs with the same seed differ:\n%+v\n%+v", first, second)
	}
}
