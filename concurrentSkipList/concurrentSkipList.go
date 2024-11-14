// package concurrentSkipList implements a concurrent skip-list leveraging lazy synchronization for atomic updates and insertions
package concurrentSkipList

import (
	"cmp"
	"context"
	"fmt"
	"github.com/RICE-COMP318-FALL24/owldb-p1group24/index_utils"
	"log/slog"
	"math/rand/v2"
	"sync"
	"sync/atomic"
)

// node is the internal data structure used by Skiplist to store key-value pairs.
type node[K cmp.Ordered, V any] struct {
	mtx sync.Mutex //used for synchronization

	fullyLinked atomic.Bool //whether the node has been fully linked to its predecessors and successors

	marked atomic.Bool //whether the node has been marked for deletion

	key K //the key the node stores

	val V //the value the node stores

	nexts []atomic.Pointer[node[K, V]] //an array of pointers to the next nodes in the skiplist

	topLevel int //the highest level list at which the node exists

	beingUpdated atomic.Bool
}

// Skiplist implements an abstract set of key-value pairs, of type K and V respectively.
// This list employs lazy synchronization for concurrent operations.
type Skiplist[K cmp.Ordered, V any] struct {
	maxK K //maximum key value

	minK K //minimum key value

	head *node[K, V] //the head of the skiplist

	mutationCount atomic.Uint64 //Updated on every successful insert & delete
}

// NewSL instantiates a skiplist, supporting keys whose value lies within the range
// minkey and maxkey (bounds exclusive). *NOTE: inserting a value with key greater than
// max will break this thing.
// Returns a pointer to the newly created skiplist
func NewSL[K cmp.Ordered, V any](minkey K, maxkey K) *Skiplist[K, V] {

	dummyHead := node[K, V]{} 

	dummyHead.key = minkey

	dummyHead.nexts = make([]atomic.Pointer[node[K, V]], 8) //prealloc saves memory and space

	dummyHead.topLevel = 8

	dummyHead.marked.Store(false)

	dummyTail := node[K, V]{}

	dummyTail.key = maxkey

	dummyTail.nexts = make([]atomic.Pointer[node[K, V]], 8) //prealloc saves memory and space

	dummyTail.marked.Store(false) 

	dummyTail.topLevel = 8 

    // Initialize the head and tail nodes
	for i := 0; i < 8; i++ {
		dummyHead.nexts[i].Store(&dummyTail) 
		dummyTail.nexts[i].Store(nil) 
	}
    // Initialize the skiplist
	var sl Skiplist[K, V] = Skiplist[K, V]{
		minK: minkey,
		maxK: maxkey,
		head: &dummyHead,
	}

	return &sl

}

// getState prints out the keys and values found at every level of the skiplist
func (sl *Skiplist[K, V]) getState() {

	for curLevel := 8 - 1; curLevel >= 0; curLevel -= 1 {

		cur := sl.head
		for cur != nil {
			slog.Debug(fmt.Sprintf("%+v (%+v) ----", cur.key, cur.val))
			cur = cur.nexts[curLevel].Load()
		}
		slog.Debug(fmt.Sprintf("<nil>"))
		slog.Debug(fmt.Sprintf("LEVEL %d", curLevel))
		slog.Debug(fmt.Sprintf("\n"))
	}
}

// find is an internal routine returning the level at which a node is found, and an array of predecessors and successors in the skipist
func (sl *Skiplist[K, V]) find(key K, preds []atomic.Pointer[node[K, V]], succs []atomic.Pointer[node[K, V]]) int {

	pred := sl.head

	levelFound := -1

	for curLevel := 8 - 1; curLevel >= 0; curLevel -= 1 {

		curr := pred.nexts[curLevel].Load()

		for key > curr.key { //Traverse until we find a node

			pred = curr

			curr = pred.nexts[curLevel].Load()
		}

		if levelFound == -1 && key == curr.key {
			slog.Debug(fmt.Sprintf("Found the node with key %+v, its highest level is %d \n", key, curLevel))
			levelFound = curLevel
		}

		preds[curLevel].Store(pred)
		succs[curLevel].Store(curr)

	}
	return levelFound

}

// randomLevel generates a random level within range for insertion into the skiplist
func randomLevel() int {
	return rand.IntN(8)
}

// Upsert either updates or inserts into the skiplist, depending on the function check's behavior (defined by the user)
// Returns true if the operation was successful, and an error if the operation failed
func (sl *Skiplist[K, V]) Upsert(key K, check index_utils.UpdateCheck[K, V]) (updated bool, err error) {

	newNodeLevel := randomLevel()

	preds := make([]atomic.Pointer[node[K, V]], 8)

	succs := make([]atomic.Pointer[node[K, V]], 8)

	for {

		levelFound := sl.find(key, preds, succs)

		if levelFound != -1 {

			foundNode := succs[levelFound].Load()

			if !foundNode.marked.Load() { //node is not marked

				for !foundNode.fullyLinked.Load() {
				}

				slog.Debug(fmt.Sprintf("A node with this key exists: %+v \n", key))

				foundNode.mtx.Lock()

				newVal, err1 := check(foundNode.key, foundNode.val, true)

				if err1 == nil { //we proceed with the update
					//Prevents other upserts from updating this value

					foundNode.val = newVal

					foundNode.mtx.Unlock()
					slog.Debug(fmt.Sprintf("OLD VALUE IS %+v \n ", foundNode.val))
					slog.Debug(fmt.Sprintf("Our updated value should be %+v \n", newVal))
					sl.mutationCount.Add(1)
					return true, nil
				} else { //we do not proceed with the update and return why
					foundNode.mtx.Unlock()
					return false, err1
				}
			}
			continue

		}

		highestLocked := -1

		var pred, succ, lastLocked *node[K, V]

		valid := true
		lockLevel := 0
		for valid && (lockLevel < 8) {

			pred = preds[lockLevel].Load()

			succ = succs[lockLevel].Load()

			if pred != lastLocked { //Can't lock the same thing twice ;)
				pred.mtx.Lock()
				slog.Debug(fmt.Sprintf("Locked the node at level %d,with value %+v \n", lockLevel, pred.key))
				highestLocked = lockLevel
				lastLocked = pred
			}

			valid = !pred.marked.Load() && !succ.marked.Load() && (pred.nexts[lockLevel].Load() == succ)
			lockLevel += 1
		}

		if !valid {

			var lastUnlocked *node[K, V]
			var predd *node[K, V]
			for i := highestLocked; i >= 0; i-- {
				predd = preds[i].Load()
				if predd != lastUnlocked {
					predd.mtx.Unlock()
					//slog.Debug(fmt.Sprintf("Unlocked the node at level %d", lockLevel))
					lastUnlocked = predd
				}
			}
			continue

		}
		var nullV V

		newVal, err2 := check(key, nullV, false)

		if err2 != nil {
			var lastUnlocked *node[K, V]
			var predd *node[K, V]
			for i := highestLocked; i >= 0; i-- {
				predd = preds[i].Load()
				if predd != lastUnlocked {
					predd.mtx.Unlock()

					lastUnlocked = predd
				}
			}
			return false, err2
		}
		newNode := &node[K, V]{

			key:      key,
			val:      newVal,
			nexts:    make([]atomic.Pointer[node[K, V]], newNodeLevel+1),
			topLevel: newNodeLevel,
		}

		for level := 0; level <= newNodeLevel; level++ {
			newNode.nexts[level].Store(succs[level].Load())
			preds[level].Load().nexts[level].Store(newNode)
		}

		newNode.fullyLinked.Store(true)
		//UNLOCKING
		var lastUnlocked *node[K, V]
		var predd *node[K, V]
		for i := highestLocked; i >= 0; i-- {
			predd = preds[i].Load()
			if predd != lastUnlocked {

				//slog.Debug(fmt.Sprintf("unlocking at level %d ,node value is %+v \n", i, predd.key))
				predd.mtx.Unlock()
				lastUnlocked = predd
			}
		}
		sl.mutationCount.Add(1)
		return true, nil

	}

}

// Find gets the value associated with key k.
// Returns the value and true if the key exists, otherwise returns the zero value of V and false
func (sl *Skiplist[K, V]) Find(key K) (V, bool) {
	preds := make([]atomic.Pointer[node[K, V]], 8)
	succs := make([]atomic.Pointer[node[K, V]], 8)
	var nullV V
	foundLevel := sl.find(key, preds, succs)
	if foundLevel == -1 {
		return nullV, false
	}
	found := succs[foundLevel].Load()
	for found.beingUpdated.Load() {
	}
	return found.val, found.fullyLinked.Load() && !found.marked.Load()
}

// Remove removes the node with key K (if it exists)
// Returns the value of the removed node and true if the node was removed,
// otherwise returns the zero value of V and false
func (sl *Skiplist[K, V]) Remove(key K) (removedVal V, removed bool) {
	var nullV V
	var victim *node[K, V]
	topLevel := -1
	isMarked := false
	for {
		preds := make([]atomic.Pointer[node[K, V]], 8)
		succs := make([]atomic.Pointer[node[K, V]], 8)
		foundLevel := sl.find(key, preds, succs)

		if foundLevel != -1 {
			victim = succs[foundLevel].Load()

		}
		if !isMarked {

			if foundLevel == -1 {
				return nullV, false
			}

			if !victim.fullyLinked.Load() {
				return nullV, false
			}

			if victim.marked.Load() {
				return nullV, false
			}

			if victim.topLevel != foundLevel {
				return nullV, false
			}

			topLevel = victim.topLevel
			//slog.Debug(fmt.Sprintf("LOCKING THE VICTIM\n"))
			victim.mtx.Lock()
			//slog.Debug(fmt.Sprintf("LOCK SUCCESS\n"))
			if victim.marked.Load() {
				victim.mtx.Unlock()
				return nullV, false
			}

			victim.marked.Store(true)
			isMarked = true
		}
		//VICTIM FOUND, LOCK PREDECESSORS
		highestLocked := -1
		//slog.Debug(fmt.Sprintf("LOCKING PREDS\n"))

		level := 0

		valid := true
		var pred *node[K, V]
		var lastLocked *node[K, V]
		for valid && (level <= topLevel) {
			pred = preds[level].Load()
			if pred != lastLocked {

				pred.mtx.Lock()
				highestLocked = level
				//slog.Debug(fmt.Sprintf("Locked node with value %+v\n", pred.key))
				lastLocked = pred
			}

			valid = !pred.marked.Load() && (pred.nexts[level].Load() == victim)
			level += 1
		}
		if !valid { //release all locks

			unlockLevel := highestLocked
			var lastUnlocked *node[K, V]
			var predd *node[K, V]
			for unlockLevel >= 0 {
				predd = preds[unlockLevel].Load()
				if lastUnlocked != predd {
					predd.mtx.Unlock()
					lastUnlocked = predd
				}
				unlockLevel -= 1
			}
			continue

		}

		unlinkLevel := topLevel

		for unlinkLevel >= 0 {
			preds[unlinkLevel].Load().nexts[unlinkLevel].Store(victim.nexts[unlinkLevel].Load())
			unlinkLevel -= 1
		}

		victim.mtx.Unlock()

		unlockLevel := highestLocked
		//slog.Debug(fmt.Sprintf("starting unlockLevel at %d\n", unlockLevel))
		var predNode *node[K, V]
		var lastUnlocked *node[K, V]
		for unlockLevel >= 0 {
			predNode = preds[unlockLevel].Load()
			if predNode != lastUnlocked {
				predNode.mtx.Unlock()

				//slog.Debug(fmt.Sprintf("removed: unlocked node with value %+v\n", pred.key))
				lastUnlocked = predNode
			}

			unlockLevel -= 1
		}
		sl.mutationCount.Add(1)
		return victim.val, true

	}
}

// Query retrieves all key-value pairs in the range [lower,upper]
// Returns a slice of index_utils.Pair[K,V] and an error if the operation fails
func (sl *Skiplist[K, V]) Query(ctx context.Context, lower K, upper K) ([]index_utils.Pair[K, V], error) {

	var lastMutationCount uint64
	slog.Debug(fmt.Sprintf("entering query, lower bound is %+v,upper bound is %+v", lower, upper))
	for { //retry until no mutations
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("the request timed out")
		default:
		}
		lastMutationCount = sl.mutationCount.Load()
		resPair := make([]index_utils.Pair[K, V], 0)
		pred := sl.head
		iterkey := sl.minK
		//slog.Debug("in here now")
		for iterkey <= upper {

			curr := pred.nexts[0].Load()
			iterkey = curr.key
			if curr.key == sl.maxK {
				break
			}
			//slog.Debug(fmt.Sprintf("current key %+v \n", iterkey))
			if ((lower <= iterkey) && (iterkey < sl.maxK) && (sl.minK < iterkey)) && curr.fullyLinked.Load() && !curr.marked.Load() {
				//slog.Debug(fmt.Sprintf("node with key %+v is in range!\n", iterkey))
				pair := index_utils.Pair[K, V]{Key: curr.key, Value: curr.val}
				resPair = append(resPair, pair)

			}
			pred = curr

			curr = pred.nexts[0].Load()
			iterkey = curr.key

		}

		mutCount := sl.mutationCount.Load()
		if mutCount == lastMutationCount {
			slog.Debug("No mutations detected")
			return resPair, nil
		}
		lastMutationCount = mutCount
		slog.Debug("A mutation was detected, we will try again")

	}

}
