package tests

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// TestStateDB allows for switching database implementation for running tests.
// It is an extension of vm.StateDB with additional methods that were originally
// available only at an implementation level of the geth database.
// Not all methods have to be available in all implementations, and clients
// should pair expected method outputs with the actual implementation.
type TestStateDB interface {
	vm.StateDB

	// Database returns the underlying database.
	Database() state.Database

	// Logs returns the logs of the current transaction.
	Logs() []*types.Log

	SetLogger(l *tracing.Hooks)

	// SetBalance sets the balance of the given account.
	SetBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason)

	// IntermediateRoot returns current state root hash.
	IntermediateRoot(deleteEmptyObjects bool) common.Hash

	// Commit commits the state to the underlying trie database and returns state root hash.
	Commit(block uint64, deleteEmptyObjects bool) (common.Hash, error)
}

// TestContextFactory is an interface for creating test configurations.
type TestContextFactory interface {

	// NewTestStateDB creates a new TestStateDB and provides a callback to set up the pre-state.
	// The implementation should create a new fresh database, apply data provided in the callback,
	// flush results and clean all temporal states.
	// In practices, a sealed state such as state root hash should be available after the call,
	// and the database should be ready for the next state transition.
	NewTestStateDB(makePreState func(db TestStateDB)) StateTestState
}

// gethFactory is a factory for creating geth database.
type gethFactory struct {
	db          ethdb.Database
	snapshotter bool
	scheme      string
}

// NewGethFactory creates a new gethFactory.
func NewGethFactory(db ethdb.Database, snapshotter bool, scheme string) TestContextFactory {
	return gethFactory{db, snapshotter, scheme}
}

// NewTestStateDB creates a new StateTestState using geth database.
func (f gethFactory) NewTestStateDB(makePreState func(db TestStateDB)) StateTestState {
	tconf := &triedb.Config{Preimages: true}
	if f.scheme == rawdb.HashScheme {
		tconf.HashDB = hashdb.Defaults
	} else {
		tconf.PathDB = pathdb.Defaults
	}

	triedb := triedb.NewDatabase(f.db, tconf)
	sdb := state.NewDatabaseWithNodeDB(f.db, triedb)
	statedb, _ := state.New(types.EmptyRootHash, sdb, nil)

	makePreState(statedb)

	// Commit and re-open to start with a clean state.
	root, _ := statedb.Commit(0, false)

	// If snapshot is requested, initialize the snapshotter and use it in state.
	var snaps *snapshot.Tree
	if f.snapshotter {
		snapconfig := snapshot.Config{
			CacheSize:  1,
			Recovery:   false,
			NoBuild:    false,
			AsyncBuild: false,
		}
		snaps, _ = snapshot.New(snapconfig, f.db, triedb, root)
	}
	statedb, _ = state.New(root, sdb, snaps)

	return StateTestState{statedb, triedb, snaps}
}
