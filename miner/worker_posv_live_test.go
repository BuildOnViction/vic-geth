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
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// Environment variable that enables the live run and sets how long each network builds blocks, e.g. "2m".
	posvLiveEnv = "POSV_LIVE_SIM"
	// Validators per live network; small enough to run dozens of networks in one process.
	posvLiveValidatorCount = 12
	// Head start a network gets to build its nodes before its genesis timestamp is reached.
	posvLiveSetupAllowance = 20 * time.Second
	// Slack added to the slowest delivery when waiting for the blocks still in flight.
	posvLiveSettleMargin = 2 * time.Second
	posvLiveInboxSize    = 256
	// Pace of the transfers that keep transactions pending; well below the shortest recommit interval.
	posvLiveTrafficInterval = 500 * time.Millisecond
	posvLiveTrafficGasPrice = 10 * params.GWei
)

var (
	posvLiveTrafficRecipient = common.HexToAddress("0x00000000000000000000000000000000000cafe1")
	posvLiveTrafficFunds     = new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(params.Ether))

	errPosvLiveSetupTooSlow = errors.New("nodes were not ready at the genesis timestamp")
	errPosvLivePastEpoch    = errors.New("run does not fit inside the first epoch")
	errPosvLiveFirstEpoch   = errors.New("engine consulted its backend inside the first epoch")
)

// Runs every scenario on real validators, next to the model's prediction for the same network. It takes the
// scenario duration of wall-clock time per batch of -parallel networks, so it is opt-in:
//
//	POSV_LIVE_SIM=2m go test ./miner -run TestPosvWaitPeriodLive -v -parallel 64
func TestPosvWaitPeriodLive(t *testing.T) {
	setting := os.Getenv(posvLiveEnv)
	if setting == "" {
		t.Skipf("runs real validators in real time; set %s to how long each network builds blocks, e.g. 2m", posvLiveEnv)
	}
	duration, err := time.ParseDuration(setting)
	if err != nil {
		t.Fatalf("%s: %v", posvLiveEnv, err)
	}
	scenarios := posvLiveScenarios(duration)
	liveResults := make([][]posvSimResult, len(scenarios))

	t.Run("networks", func(t *testing.T) {
		for i, scenario := range scenarios {
			liveResults[i] = make([]posvSimResult, len(simWaitPolicies))
			for j, policy := range simWaitPolicies {
				i, j, scenario, policy := i, j, scenario, policy
				t.Run(scenario.name+"/"+policy.name, func(t *testing.T) {
					t.Parallel()
					result, err := runPosvLive(scenario, policy, simTestSeed)
					if err != nil {
						t.Fatalf("live run failed: %v", err)
					}
					liveResults[i][j] = result

					if result.canonicalBlocks == 0 {
						t.Errorf("no block became canonical")
					}
					if result.headWeights != 1 {
						t.Errorf("online nodes ended on heads of %d weights, want 1", result.headWeights)
					}
				})
			}
		}
	})

	var table strings.Builder
	fmt.Fprintf(&table, "\n%-28s %-5s %-6s %7s %9s %9s %7s %8s %12s %7s %7s %6s\n",
		"scenario", "wait", "run", "blocks", "avg time", "worst gap", "forked", "orphaned", "lost in-turn", "backup", "reorgs", "depth")
	for i, scenario := range scenarios {
		for j, policy := range simWaitPolicies {
			writePosvLiveRow(&table, scenario.name, policy.name, "live", liveResults[i][j].asAverage())
			writePosvLiveRow(&table, "", "", "model", averagePosvSim(scenario, policy))
		}
		table.WriteString("\n")
	}
	t.Log(table.String())
}

func writePosvLiveRow(table *strings.Builder, scenario, policy, run string, average posvSimAverage) {
	fmt.Fprintf(table, "%-28s %-5s %-6s %7.1f %9s %9s %7.1f %8.1f %12.1f %7.1f %7.1f %6d\n",
		scenario, policy, run, average.canonicalBlocks,
		average.meanBlockTime.Round(time.Millisecond), average.worstBlockGap.Round(10*time.Millisecond),
		average.forkedHeights, average.orphanedBlocks, average.lostInTurnBlocks, average.backupBlocks,
		average.observerReorgs, average.deepestReorg)
}

func (result posvSimResult) asAverage() posvSimAverage {
	return posvSimAverage{
		canonicalBlocks:  float64(result.canonicalBlocks),
		meanBlockTime:    result.meanBlockTime,
		worstBlockGap:    result.worstBlockGap,
		forkedHeights:    float64(result.forkedHeights),
		orphanedBlocks:   float64(result.orphanedBlocks),
		lostInTurnBlocks: float64(result.lostInTurnBlocks),
		backupBlocks:     float64(result.backupBlocks),
		observerReorgs:   float64(result.observerReorgs),
		deepestReorg:     result.deepestReorg,
		mostHeadWeights:  result.headWeights,
	}
}

// Validator 0 stays online in every scenario: the miner lets nobody else build on the genesis block.
func posvLiveScenarios(duration time.Duration) []posvSimScenario {
	scenarios := []posvSimScenario{
		{name: "healthy, 0.3s lag", meanLag: simFastLag},
		{name: "healthy, 1.5s lag", meanLag: 1500 * time.Millisecond},
		{name: "congested, 2.5s lag", meanLag: 2500 * time.Millisecond},
		{name: "20% of blocks 2.5s late", meanLag: simFastLag, lagSpikeChance: 0.2, lagSpike: 2500 * time.Millisecond},
		{name: "1 offline", meanLag: simFastLag, offline: []int{5}},
		{name: "1 offline, 1s lag", meanLag: time.Second, offline: []int{5}},
		{name: "2 consecutive offline", meanLag: simFastLag, offline: []int{5, 6}},
		{name: "1 validator 3s slower", meanLag: simFastLag, slow: map[int]time.Duration{5: 3 * time.Second}},
		{name: "healthy 1s lag, recommit 1s", meanLag: time.Second, recommit: time.Second},
		{name: "1 offline, empty tx pool", meanLag: simFastLag, offline: []int{5}, hasEmptyTxPool: true},
	}
	for i := range scenarios {
		scenarios[i].validatorCount = posvLiveValidatorCount
		scenarios[i].duration = duration
	}
	return scenarios
}

func runPosvLive(scenario posvSimScenario, policy posvWaitPolicy, seed int64) (posvSimResult, error) {
	network, err := newPosvLiveNetwork(scenario, policy, seed)
	if err != nil {
		return posvSimResult{}, err
	}
	defer network.close()
	if err := network.run(); err != nil {
		return posvSimResult{}, err
	}
	return network.result()
}

// posvLiveNetwork runs one real PoSV engine, miner worker and blockchain per validator, in real time. Blocks
// travel between them through the simNetwork the model uses, in place of devp2p and the block fetcher. The chain
// stays inside its first epoch, where blocks carry no attestor signature: attestation exists only as that
// network's extra hop and its refusal to deliver a block whose attestor is offline.
type posvLiveNetwork struct {
	scenario    posvSimScenario
	policy      posvWaitPolicy
	genesisTime time.Time
	// Validators by index, then the observer; nil where a validator is offline.
	nodes []*liveNode
	// Funded account whose transfers keep transactions pending on every validator.
	trafficKey *ecdsa.PrivateKey

	quit    chan struct{}
	running sync.WaitGroup

	mu      sync.Mutex
	network *simNetwork
	start   *simBlock
	// Every block that was delivered, by hash, with the genesis block as start.
	records  map[common.Hash]*simBlock
	attested []*simBlock
	failure  error
}

type liveNode struct {
	index int
	chain *core.BlockChain
	// Nil on the observer, as are mux and worker.
	txPool *core.TxPool
	mux    *event.TypeMux
	worker *worker
	inbox  chan *types.Block

	// Touched only by the node's import goroutine.
	awaitingParent map[common.Hash][]*types.Block
	// Guarded by posvLiveNetwork.mu.
	reorgCount   int
	deepestReorg int
}

func newPosvLiveNetwork(scenario posvSimScenario, policy posvWaitPolicy, seed int64) (*posvLiveNetwork, error) {
	scenario = scenario.withDefaults()
	config := params.VictionChainConfig
	// Block epoch-gap makes the chain read the validator contract, which this genesis lacks.
	if uint64(scenario.duration/simBlockPeriod)+uint64(scenario.validatorCount) >= config.Posv.Epoch-config.Posv.Gap {
		return nil, errPosvLivePastEpoch
	}
	keys := make([]*ecdsa.PrivateKey, scenario.validatorCount)
	extra := make([]byte, posv.ExtraVanity)
	for i := range keys {
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("generating validator key: %w", err)
		}
		keys[i] = key
		extra = append(extra, crypto.PubkeyToAddress(key.PublicKey).Bytes()...)
	}
	extra = append(extra, make([]byte, posv.ExtraSeal)...)
	trafficKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("generating traffic key: %w", err)
	}

	network := &posvLiveNetwork{
		scenario:    scenario,
		policy:      policy,
		genesisTime: time.Now().Add(posvLiveSetupAllowance).Truncate(time.Second),
		trafficKey:  trafficKey,
		quit:        make(chan struct{}),
		network:     newSimNetwork(scenario, rand.New(rand.NewSource(seed))),
		start:       &simBlock{distance: 1},
		records:     map[common.Hash]*simBlock{},
	}
	genesis := &core.Genesis{
		Config:    config,
		Timestamp: uint64(network.genesisTime.Unix()),
		ExtraData: extra,
		GasLimit:  params.VictionGasLimit,
		Alloc:     core.GenesisAlloc{crypto.PubkeyToAddress(trafficKey.PublicKey): {Balance: posvLiveTrafficFunds}},
	}
	for index := 0; index <= scenario.validatorCount; index++ {
		if network.network.isOffline[index] {
			network.nodes = append(network.nodes, nil)
			continue
		}
		var key *ecdsa.PrivateKey
		if index < scenario.validatorCount {
			key = keys[index]
		}
		node, err := newLiveNode(index, genesis, key, scenario, policy)
		if err != nil {
			network.close()
			return nil, fmt.Errorf("creating node %d: %w", index, err)
		}
		network.nodes = append(network.nodes, node)
		network.records[node.chain.Genesis().Hash()] = network.start
	}
	return network, nil
}

func newLiveNode(index int, genesis *core.Genesis, key *ecdsa.PrivateKey, scenario posvSimScenario, policy posvWaitPolicy) (*liveNode, error) {
	db := rawdb.NewMemoryDatabase()
	if _, err := genesis.Commit(db); err != nil {
		return nil, fmt.Errorf("committing genesis: %w", err)
	}
	engine := posv.New(genesis.Config.Posv, db)
	engine.SetBackend(liveFirstEpochBackend{})

	// Mirrors Ethereum.shouldPreserve: a validator keeps its own block on a total difficulty tie.
	var isLocalBlock func(*types.Block) bool
	if key != nil {
		address := crypto.PubkeyToAddress(key.PublicKey)
		isLocalBlock = func(block *types.Block) bool {
			author, err := engine.Author(block.Header())
			return err == nil && author == address
		}
	}
	chain, err := core.NewBlockChain(db, &core.CacheConfig{TrieDirtyDisabled: true}, genesis.Config, engine, vm.Config{}, isLocalBlock, nil)
	if err != nil {
		return nil, fmt.Errorf("creating blockchain: %w", err)
	}
	node := &liveNode{
		index:          index,
		chain:          chain,
		inbox:          make(chan *types.Block, posvLiveInboxSize),
		awaitingParent: map[common.Hash][]*types.Block{},
	}
	if key == nil {
		return node, nil
	}
	address := crypto.PubkeyToAddress(key.PublicKey)
	engine.Authorize(address, func(_ accounts.Account, _ string, message []byte) ([]byte, error) {
		return crypto.Sign(crypto.Keccak256(message), key)
	})
	node.txPool = core.NewTxPool(testTxPoolConfig, genesis.Config, chain)
	node.mux = new(event.TypeMux)
	backend := &testWorkerBackend{db: db, chain: chain, txPool: node.txPool}
	minerConfig := &Config{Recommit: scenario.recommit, GasCeil: genesis.GasLimit}
	node.worker = newWorker(minerConfig, genesis.Config, engine, backend, node.mux, isLocalBlock, false)
	node.worker.posvWaitPeriod = policy.waitPeriod
	node.worker.posvWaitPeriodCheckpoint = policy.waitPeriodCheckpoint
	node.worker.setEtherbase(address)
	return node, nil
}

// Starts the validators at the genesis timestamp, lets them build for the scenario's duration, then waits for
// the blocks still in flight.
func (net *posvLiveNetwork) run() error {
	for _, node := range net.nodes {
		if node == nil {
			continue
		}
		net.running.Add(1)
		go net.importLoop(node)
		if node.worker != nil {
			net.running.Add(1)
			go net.broadcastLoop(node, node.mux.Subscribe(core.NewMinedBlockEvent{}))
		}
	}
	if time.Now().After(net.genesisTime) {
		return errPosvLiveSetupTooSlow
	}
	time.Sleep(time.Until(net.genesisTime))
	if !net.scenario.hasEmptyTxPool {
		net.running.Add(1)
		go net.trafficLoop()
	}
	for _, node := range net.nodes {
		if node != nil && node.worker != nil {
			node.worker.start()
		}
	}
	time.Sleep(time.Until(net.genesisTime.Add(net.scenario.duration)))
	for _, node := range net.nodes {
		if node != nil && node.worker != nil {
			node.worker.stop()
		}
	}
	time.Sleep(net.settleTime())
	return nil
}

// Returns the time a block sealed just before the validators stopped needs to be released and reach every node.
func (net *posvLiveNetwork) settleTime() time.Duration {
	var slowest time.Duration
	for _, extra := range net.scenario.slow {
		if extra > slowest {
			slowest = extra
		}
	}
	return simBlockPeriod + net.scenario.meanLag*3/2 + net.scenario.lagSpike + 4*slowest + posvLiveSettleMargin
}

func (net *posvLiveNetwork) close() {
	close(net.quit)
	for _, node := range net.nodes {
		if node == nil {
			continue
		}
		if node.worker != nil {
			node.mux.Stop()
			node.worker.close()
			node.txPool.Stop()
		}
		node.chain.Stop()
	}
	net.running.Wait()
}

func (net *posvLiveNetwork) fail(err error) {
	net.mu.Lock()
	defer net.mu.Unlock()
	if net.failure == nil {
		net.failure = err
	}
}

// Hands every validator the same transfer at a steady pace, standing in for the traffic of a live network: on a
// recommit tick the miner builds only when transactions are pending.
func (net *posvLiveNetwork) trafficLoop() {
	defer net.running.Done()
	ticker := time.NewTicker(posvLiveTrafficInterval)
	defer ticker.Stop()
	for nonce := uint64(0); ; nonce++ {
		select {
		case <-ticker.C:
		case <-net.quit:
			return
		}
		transfer := types.NewTransaction(nonce, posvLiveTrafficRecipient, common.Big1, params.TxGas, big.NewInt(posvLiveTrafficGasPrice), nil)
		// Unprotected signature: valid from block 1, before the chainspec activates EIP-155.
		tx, err := types.SignTx(transfer, types.HomesteadSigner{}, net.trafficKey)
		if err != nil {
			net.fail(fmt.Errorf("signing traffic transaction %d: %w", nonce, err))
			return
		}
		for _, node := range net.nodes {
			if node == nil || node.txPool == nil {
				continue
			}
			if err := node.txPool.AddRemote(tx); err != nil {
				net.fail(fmt.Errorf("node %d rejected traffic transaction %d: %w", node.index, nonce, err))
				return
			}
		}
	}
}

func (net *posvLiveNetwork) broadcastLoop(node *liveNode, mined *event.TypeMuxSubscription) {
	defer net.running.Done()
	for minedEvent := range mined.Chan() {
		block := minedEvent.Data.(core.NewMinedBlockEvent).Block
		for _, delivery := range net.release(node.index, block) {
			net.deliverAfter(delivery.delay, net.nodes[delivery.node], block)
		}
	}
}

// Records a sealed block and decides, through the fake network, who receives it and when.
func (net *posvLiveNetwork) release(creator int, block *types.Block) []simDelivery {
	releasedAt := time.Since(net.genesisTime)
	net.mu.Lock()
	defer net.mu.Unlock()

	if _, isKnown := net.records[block.Hash()]; isKnown {
		return nil
	}
	parent := net.records[block.ParentHash()]
	if parent == nil {
		if net.failure == nil {
			net.failure = fmt.Errorf("validator %d built block %d on a parent that was never delivered", creator, block.NumberU64())
		}
		return nil
	}
	deliveries, isAttested := net.network.deliveries(creator, block.NumberU64())
	if !isAttested && parent == net.start {
		// Miner gives the first block no backup, so losing it to an offline attestor would stall the run.
		deliveries, isAttested = net.network.deliveriesVia(creator, creator)
	}
	if !isAttested {
		return nil
	}
	record := &simBlock{
		parent:     parent,
		number:     block.NumberU64(),
		creator:    creator,
		timestamp:  int64(block.Time()),
		distance:   net.scenario.validatorCount - int(block.Difficulty().Int64()) + 1,
		releasedAt: releasedAt,
	}
	net.records[block.Hash()] = record
	net.attested = append(net.attested, record)
	return deliveries
}

func (net *posvLiveNetwork) deliverAfter(delay time.Duration, node *liveNode, block *types.Block) {
	net.running.Add(1)
	go func() {
		defer net.running.Done()
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-net.quit:
			return
		}
		select {
		case node.inbox <- block:
		case <-net.quit:
		}
	}()
}

func (net *posvLiveNetwork) importLoop(node *liveNode) {
	defer net.running.Done()
	for {
		select {
		case block := <-node.inbox:
			net.importBlock(node, block)
		case <-net.quit:
			return
		}
	}
}

// Inserts a delivered block into the node's chain, holding it back until its parent has arrived.
func (net *posvLiveNetwork) importBlock(node *liveNode, block *types.Block) {
	if node.chain.HasBlock(block.Hash(), block.NumberU64()) {
		return
	}
	if !node.chain.HasBlock(block.ParentHash(), block.NumberU64()-1) {
		node.awaitingParent[block.ParentHash()] = append(node.awaitingParent[block.ParentHash()], block)
		return
	}
	oldHead := node.chain.CurrentBlock().Hash()
	if _, err := node.chain.InsertChain(types.Blocks{block}); err != nil {
		net.fail(fmt.Errorf("node %d importing block %d: %w", node.index, block.NumberU64(), err))
		return
	}
	if newHead := node.chain.CurrentBlock().Hash(); newHead != oldHead {
		net.recordHeadSwitch(node, oldHead, newHead)
	}
	children := node.awaitingParent[block.Hash()]
	delete(node.awaitingParent, block.Hash())
	for _, child := range children {
		net.importBlock(node, child)
	}
}

func (net *posvLiveNetwork) recordHeadSwitch(node *liveNode, oldHead, newHead common.Hash) {
	net.mu.Lock()
	defer net.mu.Unlock()
	dropped := droppedBlockCount(net.records[oldHead], net.records[newHead])
	if dropped == 0 {
		return
	}
	node.reorgCount++
	if dropped > node.deepestReorg {
		node.deepestReorg = dropped
	}
}

func (net *posvLiveNetwork) result() (posvSimResult, error) {
	net.mu.Lock()
	defer net.mu.Unlock()
	if net.failure != nil {
		return posvSimResult{}, net.failure
	}
	observer := net.nodes[net.scenario.validatorCount]
	headWeights := map[string]struct{}{}
	for _, node := range net.nodes {
		if node != nil {
			head := node.chain.CurrentBlock()
			headWeights[node.chain.GetTd(head.Hash(), head.NumberU64()).String()] = struct{}{}
		}
	}
	return posvRun{
		validatorCount: net.scenario.validatorCount,
		start:          net.start,
		attested:       net.attested,
		observerHead:   net.records[observer.chain.CurrentBlock().Hash()],
		observerReorgs: observer.reorgCount,
		deepestReorg:   observer.deepestReorg,
		headWeights:    len(headWeights),
	}.summarize(), nil
}

// liveFirstEpochBackend satisfies posv.PosvBackend for a chain inside its first epoch, where the engine needs a
// backend to be set but consults it for neither attestors nor checkpoints.
type liveFirstEpochBackend struct{}

func (liveFirstEpochBackend) PosvGetAttestors(*types.Header, []common.Address, *params.VictionConfig) ([]int64, error) {
	return nil, errPosvLiveFirstEpoch
}

func (liveFirstEpochBackend) PosvGetCreatorAttestorPairs(
	_, _ *types.Header, _ *params.ChainConfig, _ *params.PosvConfig, _ *params.VictionConfig,
) (map[common.Address]common.Address, uint64, error) {
	return nil, 0, errPosvLiveFirstEpoch
}

func (liveFirstEpochBackend) PosvGetEpochRewards(
	*types.Header, *params.ChainConfig, *params.PosvConfig, *params.VictionConfig, consensus.ChainReader, *state.StateDB, log.Logger,
) (*posv.EpochReward, error) {
	return nil, errPosvLiveFirstEpoch
}

func (liveFirstEpochBackend) PosvDistributeEpochRewards(*types.Header, *posv.EpochReward, *state.StateDB) error {
	return errPosvLiveFirstEpoch
}

func (liveFirstEpochBackend) PosvGetPenalties(
	*types.Header, []common.Address, *params.ChainConfig, *params.PosvConfig, *params.VictionConfig, consensus.ChainReader,
) ([]common.Address, error) {
	return nil, errPosvLiveFirstEpoch
}

func (liveFirstEpochBackend) PosvGetValidators(
	*types.Header, *params.ChainConfig, *params.VictionConfig, consensus.ChainReader,
) ([]common.Address, error) {
	return nil, errPosvLiveFirstEpoch
}
