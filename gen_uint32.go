// Code generated by gen.go; DO NOT EDIT.

package skipmap

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// Uint32Map represents a map based on skip list.
type Uint32Map[valueT any] struct {
	length       int64
	highestLevel uint64 // highest level for now
	header       *uint32node[valueT]
}

type uint32node[valueT any] struct {
	value unsafe.Pointer // *any
	flags bitflag
	key   uint32
	next  optionalArray // [level]*uint32node
	mu    sync.Mutex
	level uint32
}

func newUint32Node[valueT any](key uint32, value valueT, level int) *uint32node[valueT] {
	node := &uint32node[valueT]{
		key:   key,
		level: uint32(level),
	}
	node.storeVal(value)
	if level > op1 {
		node.next.extra = new([op2]unsafe.Pointer)
	}
	return node
}

func (n *uint32node[valueT]) storeVal(value valueT) {
	atomic.StorePointer(&n.value, unsafe.Pointer(&value))
}

func (n *uint32node[valueT]) loadVal() valueT {
	return *(*valueT)(atomic.LoadPointer(&n.value))
}

func (n *uint32node[valueT]) loadNext(i int) *uint32node[valueT] {
	return (*uint32node[valueT])(n.next.load(i))
}

func (n *uint32node[valueT]) storeNext(i int, node *uint32node[valueT]) {
	n.next.store(i, unsafe.Pointer(node))
}

func (n *uint32node[valueT]) atomicLoadNext(i int) *uint32node[valueT] {
	return (*uint32node[valueT])(n.next.atomicLoad(i))
}

func (n *uint32node[valueT]) atomicStoreNext(i int, node *uint32node[valueT]) {
	n.next.atomicStore(i, unsafe.Pointer(node))
}

// findNode takes a key and two maximal-height arrays then searches exactly as in a sequential skipmap.
// The returned preds and succs always satisfy preds[i] > key >= succs[i].
// (without fullpath, if find the node will return immediately)
func (s *Uint32Map[valueT]) findNode(key uint32, preds *[maxLevel]*uint32node[valueT], succs *[maxLevel]*uint32node[valueT]) *uint32node[valueT] {
	x := s.header
	for i := int(atomic.LoadUint64(&s.highestLevel)) - 1; i >= 0; i-- {
		succ := x.atomicLoadNext(i)
		for succ != nil && (succ.key < key) {
			x = succ
			succ = x.atomicLoadNext(i)
		}
		preds[i] = x
		succs[i] = succ

		// Check if the key already in the skipmap.
		if succ != nil && succ.key == key {
			return succ
		}
	}
	return nil
}

// findNodeDelete takes a key and two maximal-height arrays then searches exactly as in a sequential skip-list.
// The returned preds and succs always satisfy preds[i] > key >= succs[i].
func (s *Uint32Map[valueT]) findNodeDelete(key uint32, preds *[maxLevel]*uint32node[valueT], succs *[maxLevel]*uint32node[valueT]) int {
	// lFound represents the index of the first layer at which it found a node.
	lFound, x := -1, s.header
	for i := int(atomic.LoadUint64(&s.highestLevel)) - 1; i >= 0; i-- {
		succ := x.atomicLoadNext(i)
		for succ != nil && (succ.key < key) {
			x = succ
			succ = x.atomicLoadNext(i)
		}
		preds[i] = x
		succs[i] = succ

		// Check if the key already in the skip list.
		if lFound == -1 && succ != nil && succ.key == key {
			lFound = i
		}
	}
	return lFound
}

func unlockuint32[valueT any](preds [maxLevel]*uint32node[valueT], highestLevel int) {
	var prevPred *uint32node[valueT]
	for i := highestLevel; i >= 0; i-- {
		if preds[i] != prevPred { // the node could be unlocked by previous loop
			preds[i].mu.Unlock()
			prevPred = preds[i]
		}
	}
}

// Store sets the value for a key.
func (s *Uint32Map[valueT]) Store(key uint32, value valueT) {
	level := s.randomlevel()
	var preds, succs [maxLevel]*uint32node[valueT]
	for {
		nodeFound := s.findNode(key, &preds, &succs)
		if nodeFound != nil { // indicating the key is already in the skip-list
			if !nodeFound.flags.Get(marked) {
				// We don't need to care about whether or not the node is fully linked,
				// just replace the value.
				nodeFound.storeVal(value)
				return
			}
			// If the node is marked, represents some other goroutines is in the process of deleting this node,
			// we need to add this node in next loop.
			continue
		}

		// Add this node into skip list.
		var (
			highestLocked        = -1 // the highest level being locked by this process
			valid                = true
			pred, succ, prevPred *uint32node[valueT]
		)
		for layer := 0; valid && layer < level; layer++ {
			pred = preds[layer]   // target node's previous node
			succ = succs[layer]   // target node's next node
			if pred != prevPred { // the node in this layer could be locked by previous loop
				pred.mu.Lock()
				highestLocked = layer
				prevPred = pred
			}
			// valid check if there is another node has inserted into the skip list in this layer during this process.
			// It is valid if:
			// 1. The previous node and next node both are not marked.
			// 2. The previous node's next node is succ in this layer.
			valid = !pred.flags.Get(marked) && (succ == nil || !succ.flags.Get(marked)) && pred.loadNext(layer) == succ
		}
		if !valid {
			unlockuint32(preds, highestLocked)
			continue
		}

		nn := newUint32Node(key, value, level)
		for layer := 0; layer < level; layer++ {
			nn.storeNext(layer, succs[layer])
			preds[layer].atomicStoreNext(layer, nn)
		}
		nn.flags.SetTrue(fullyLinked)
		unlockuint32(preds, highestLocked)
		atomic.AddInt64(&s.length, 1)
	}
}

func (s *Uint32Map[valueT]) randomlevel() int {
	// Generate random level.
	level := randomLevel()
	// Update highest level if possible.
	for {
		hl := atomic.LoadUint64(&s.highestLevel)
		if uint64(level) <= hl {
			break
		}
		if atomic.CompareAndSwapUint64(&s.highestLevel, hl, uint64(level)) {
			break
		}
	}
	return level
}

// Load returns the value stored in the map for a key, or nil if no
// value is present.
// The ok result indicates whether value was found in the map.
func (s *Uint32Map[valueT]) Load(key uint32) (value valueT, ok bool) {
	x := s.header
	for i := int(atomic.LoadUint64(&s.highestLevel)) - 1; i >= 0; i-- {
		nex := x.atomicLoadNext(i)
		for nex != nil && (nex.key < key) {
			x = nex
			nex = x.atomicLoadNext(i)
		}

		// Check if the key already in the skip list.
		if nex != nil && nex.key == key {
			if nex.flags.MGet(fullyLinked|marked, fullyLinked) {
				return nex.loadVal(), true
			}
			return
		}
	}
	return
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
// (Modified from Delete)
func (s *Uint32Map[valueT]) LoadAndDelete(key uint32) (value valueT, loaded bool) {
	var (
		nodeToDelete *uint32node[valueT]
		isMarked     bool // represents if this operation mark the node
		topLayer     = -1
		preds, succs [maxLevel]*uint32node[valueT]
	)
	for {
		lFound := s.findNodeDelete(key, &preds, &succs)
		if isMarked || // this process mark this node or we can find this node in the skip list
			lFound != -1 && succs[lFound].flags.MGet(fullyLinked|marked, fullyLinked) && (int(succs[lFound].level)-1) == lFound {
			if !isMarked { // we don't mark this node for now
				nodeToDelete = succs[lFound]
				topLayer = lFound
				nodeToDelete.mu.Lock()
				if nodeToDelete.flags.Get(marked) {
					// The node is marked by another process,
					// the physical deletion will be accomplished by another process.
					nodeToDelete.mu.Unlock()
					return
				}
				nodeToDelete.flags.SetTrue(marked)
				isMarked = true
			}
			// Accomplish the physical deletion.
			var (
				highestLocked        = -1 // the highest level being locked by this process
				valid                = true
				pred, succ, prevPred *uint32node[valueT]
			)
			for layer := 0; valid && (layer <= topLayer); layer++ {
				pred, succ = preds[layer], succs[layer]
				if pred != prevPred { // the node in this layer could be locked by previous loop
					pred.mu.Lock()
					highestLocked = layer
					prevPred = pred
				}
				// valid check if there is another node has inserted into the skip list in this layer
				// during this process, or the previous is deleted by another process.
				// It is valid if:
				// 1. the previous node exists.
				// 2. no another node has inserted into the skip list in this layer.
				valid = !pred.flags.Get(marked) && pred.loadNext(layer) == succ
			}
			if !valid {
				unlockuint32(preds, highestLocked)
				continue
			}
			for i := topLayer; i >= 0; i-- {
				// Now we own the `nodeToDelete`, no other goroutine will modify it.
				// So we don't need `nodeToDelete.loadNext`
				preds[i].atomicStoreNext(i, nodeToDelete.loadNext(i))
			}
			nodeToDelete.mu.Unlock()
			unlockuint32(preds, highestLocked)
			atomic.AddInt64(&s.length, -1)
			return nodeToDelete.loadVal(), true
		}
		return
	}
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
// (Modified from Store)
func (s *Uint32Map[valueT]) LoadOrStore(key uint32, value valueT) (actual valueT, loaded bool) {
	level := s.randomlevel()
	var preds, succs [maxLevel]*uint32node[valueT]
	for {
		nodeFound := s.findNode(key, &preds, &succs)
		if nodeFound != nil { // indicating the key is already in the skip-list
			if !nodeFound.flags.Get(marked) {
				// We don't need to care about whether or not the node is fully linked,
				// just return the value.
				return nodeFound.loadVal(), true
			}
			// If the node is marked, represents some other goroutines is in the process of deleting this node,
			// we need to add this node in next loop.
			continue
		}

		// Add this node into skip list.
		var (
			highestLocked        = -1 // the highest level being locked by this process
			valid                = true
			pred, succ, prevPred *uint32node[valueT]
		)
		for layer := 0; valid && layer < level; layer++ {
			pred = preds[layer]   // target node's previous node
			succ = succs[layer]   // target node's next node
			if pred != prevPred { // the node in this layer could be locked by previous loop
				pred.mu.Lock()
				highestLocked = layer
				prevPred = pred
			}
			// valid check if there is another node has inserted into the skip list in this layer during this process.
			// It is valid if:
			// 1. The previous node and next node both are not marked.
			// 2. The previous node's next node is succ in this layer.
			valid = !pred.flags.Get(marked) && (succ == nil || !succ.flags.Get(marked)) && pred.loadNext(layer) == succ
		}
		if !valid {
			unlockuint32(preds, highestLocked)
			continue
		}

		nn := newUint32Node(key, value, level)
		for layer := 0; layer < level; layer++ {
			nn.storeNext(layer, succs[layer])
			preds[layer].atomicStoreNext(layer, nn)
		}
		nn.flags.SetTrue(fullyLinked)
		unlockuint32(preds, highestLocked)
		atomic.AddInt64(&s.length, 1)
		return value, false
	}
}

// LoadOrStoreLazy returns the existing value for the key if present.
// Otherwise, it stores and returns the given value from f, f will only be called once.
// The loaded result is true if the value was loaded, false if stored.
// (Modified from LoadOrStore)
func (s *Uint32Map[valueT]) LoadOrStoreLazy(key uint32, f func() valueT) (actual valueT, loaded bool) {
	level := s.randomlevel()
	var preds, succs [maxLevel]*uint32node[valueT]
	for {
		nodeFound := s.findNode(key, &preds, &succs)
		if nodeFound != nil { // indicating the key is already in the skip-list
			if !nodeFound.flags.Get(marked) {
				// We don't need to care about whether or not the node is fully linked,
				// just return the value.
				return nodeFound.loadVal(), true
			}
			// If the node is marked, represents some other goroutines is in the process of deleting this node,
			// we need to add this node in next loop.
			continue
		}

		// Add this node into skip list.
		var (
			highestLocked        = -1 // the highest level being locked by this process
			valid                = true
			pred, succ, prevPred *uint32node[valueT]
		)
		for layer := 0; valid && layer < level; layer++ {
			pred = preds[layer]   // target node's previous node
			succ = succs[layer]   // target node's next node
			if pred != prevPred { // the node in this layer could be locked by previous loop
				pred.mu.Lock()
				highestLocked = layer
				prevPred = pred
			}
			// valid check if there is another node has inserted into the skip list in this layer during this process.
			// It is valid if:
			// 1. The previous node and next node both are not marked.
			// 2. The previous node's next node is succ in this layer.
			valid = !pred.flags.Get(marked) && pred.loadNext(layer) == succ && (succ == nil || !succ.flags.Get(marked))
		}
		if !valid {
			unlockuint32(preds, highestLocked)
			continue
		}
		value := f()
		nn := newUint32Node(key, value, level)
		for layer := 0; layer < level; layer++ {
			nn.storeNext(layer, succs[layer])
			preds[layer].atomicStoreNext(layer, nn)
		}
		nn.flags.SetTrue(fullyLinked)
		unlockuint32(preds, highestLocked)
		atomic.AddInt64(&s.length, 1)
		return value, false
	}
}

// Delete deletes the value for a key.
func (s *Uint32Map[valueT]) Delete(key uint32) bool {
	var (
		nodeToDelete *uint32node[valueT]
		isMarked     bool // represents if this operation mark the node
		topLayer     = -1
		preds, succs [maxLevel]*uint32node[valueT]
	)
	for {
		lFound := s.findNodeDelete(key, &preds, &succs)
		if isMarked || // this process mark this node or we can find this node in the skip list
			lFound != -1 && succs[lFound].flags.MGet(fullyLinked|marked, fullyLinked) && (int(succs[lFound].level)-1) == lFound {
			if !isMarked { // we don't mark this node for now
				nodeToDelete = succs[lFound]
				topLayer = lFound
				nodeToDelete.mu.Lock()
				if nodeToDelete.flags.Get(marked) {
					// The node is marked by another process,
					// the physical deletion will be accomplished by another process.
					nodeToDelete.mu.Unlock()
					return false
				}
				nodeToDelete.flags.SetTrue(marked)
				isMarked = true
			}
			// Accomplish the physical deletion.
			var (
				highestLocked        = -1 // the highest level being locked by this process
				valid                = true
				pred, succ, prevPred *uint32node[valueT]
			)
			for layer := 0; valid && (layer <= topLayer); layer++ {
				pred, succ = preds[layer], succs[layer]
				if pred != prevPred { // the node in this layer could be locked by previous loop
					pred.mu.Lock()
					highestLocked = layer
					prevPred = pred
				}
				// valid check if there is another node has inserted into the skip list in this layer
				// during this process, or the previous is deleted by another process.
				// It is valid if:
				// 1. the previous node exists.
				// 2. no another node has inserted into the skip list in this layer.
				valid = !pred.flags.Get(marked) && pred.atomicLoadNext(layer) == succ
			}
			if !valid {
				unlockuint32(preds, highestLocked)
				continue
			}
			for i := topLayer; i >= 0; i-- {
				// Now we own the `nodeToDelete`, no other goroutine will modify it.
				// So we don't need `nodeToDelete.loadNext`
				preds[i].atomicStoreNext(i, nodeToDelete.loadNext(i))
			}
			nodeToDelete.mu.Unlock()
			unlockuint32(preds, highestLocked)
			atomic.AddInt64(&s.length, -1)
			return true
		}
		return false
	}
}

// Range calls f sequentially for each key and value present in the skipmap.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently, Range may reflect any mapping for that key
// from any point during the Range call.
func (s *Uint32Map[valueT]) Range(f func(key uint32, value valueT) bool) {
	x := s.header.atomicLoadNext(0)
	for x != nil {
		if !x.flags.MGet(fullyLinked|marked, fullyLinked) {
			x = x.atomicLoadNext(0)
			continue
		}
		if !f(x.key, x.loadVal()) {
			break
		}
		x = x.atomicLoadNext(0)
	}
}

// RangeFrom calls f sequentially for each key >= `start` and value present in the ascending skipmap,
// and for each key <= `start` and value present in the descending skipmap.
// If f returns false, range stops the iteration. If `start` is not in the skipmap, the iteration
// starts from the first key that is greater than `start` for ascending skipmap and smaller than
// `start` for descending skipmap.
//
// RangeFrom does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently, Range may reflect any mapping for that key
// from any point during the Range call.
func (s *Uint32Map[valueT]) RangeFrom(start uint32, f func(key uint32, value valueT) bool) {
	var preds, succs [maxLevel]*uint32node[valueT]
	x := s.findNode(start, &preds, &succs)
	// preds[i].key < key <= succs[i].key for ascending skipmap
	// preds[i].key > key >= succs[i].key for descending skipmap

	// x doesn't exists in the map; use the cloest node
	if x == nil {
		x = succs[0]
	}
	for x != nil {
		if !x.flags.MGet(fullyLinked|marked, fullyLinked) {
			x = x.atomicLoadNext(0)
			continue
		}
		if !f(x.key, x.loadVal()) {
			break
		}
		x = x.atomicLoadNext(0)
	}
}

// Len returns the length of this skipmap.
func (s *Uint32Map[valueT]) Len() int {
	return int(atomic.LoadInt64(&s.length))
}
