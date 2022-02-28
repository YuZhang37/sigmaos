package threadmgr

import (
	"sync"

	//	"github.com/sasha-s/go-deadlock"

	np "ulambda/ninep"
)

/*
 * The ThreadMgr struct ensures that only one operation on the thread is being
 * run at any one time. It assumes that any condition variables which are
 * passed into it are only being used/slept on by one goroutine. It relies on
 * the locking behavior of sessconds in the layer above it to ensure that there
 * aren't any sleep/wake races. Sessconds' locking behavior should be changed
 * with care.
 */

type ThreadMgr struct {
	sync.Locker
	newOpCond *sync.Cond // Signalled when there are new ops available.
	cond      *sync.Cond // Signalled when the currently running op is about to sleep or terminate. This indicates that the thread can continue running.
	done      bool
	ops       []*Op        // List of (new) ops to process.
	wakeups   []*sync.Cond // List of conds (goroutines/ops) that need to wake up.
	executing map[*Op]bool // Map of currently executing ops.
	numops    uint64       // Number of ops processed or currently processing.
	pfn       ProcessFn
}

func makeThreadMgr(pfn ProcessFn) *ThreadMgr {
	t := &ThreadMgr{}
	t.Locker = &sync.Mutex{}
	t.newOpCond = sync.NewCond(t.Locker)
	t.cond = sync.NewCond(t.Locker)
	t.ops = []*Op{}
	t.wakeups = []*sync.Cond{}
	t.executing = make(map[*Op]bool)
	t.pfn = pfn
	return t
}

func (t *ThreadMgr) GetExecuting() map[*Op]bool {
	return t.executing
}

// Enqueue a new op to be processed.
func (t *ThreadMgr) Process(fc *np.Fcall, replies chan *np.Fcall) {
	t.Lock()
	defer t.Unlock()

	t.numops++
	op := makeOp(fc, replies, t.numops)
	t.ops = append(t.ops, op)
	t.executing[op] = true

	// Signal that there are new ops to be processed.
	t.newOpCond.Signal()
}

// Notify the ThreadMgr that a goroutine/operation would like to go to sleep.
func (t *ThreadMgr) Sleep(c *sync.Cond) {

	// Acquire the *t.Mutex to ensure that the ThreadMgr.run thread won't miss the
	// sleep signal.
	t.Lock()

	// Notify the thread that this goroutine is going to sleep.
	t.cond.Signal()

	// Allow the thread to make progress, as this goroutine is about to sleep.
	// Sleep/wake races here are avoided by SessCond locking above us.
	t.Unlock()

	// Sleep on this condition variable.
	c.Wait()
}

// Called when an operation is going to be woken up. The caller may be a
// goroutine/operation belonging to this thread/session or the
// goroutine/operation may belong to another thread/session.
func (t *ThreadMgr) Wake(c *sync.Cond) {
	t.Lock()
	defer t.Unlock()

	// Append to the list of goroutines to be woken up.
	t.wakeups = append(t.wakeups, c)

	// Notify ThreadMgr that there are more wakeups to be processed. This signal
	// will be missed if the thread is not currently waiting for new ops (in the
	// tight loop in ThreadMgr.run). However, this is ok, as it checks for any new
	// wakeups that need to be processed before calling newOpCond.Wait() again.
	t.newOpCond.Signal()
}

// Processes operations on a single channel.
func (t *ThreadMgr) run() {
	t.Lock()
	for {
		// Check if we have pending wakeups or new ops.
		for len(t.ops) == 0 && len(t.wakeups) == 0 {
			if t.done {
				return
			}

			// Wait until we have new wakeups or new ops.
			t.newOpCond.Wait()
		}

		// We always process wakeups if there are any pending, and only process new
		// ops if there are no wakeups left. This is important for replication: a
		// wakeup in the previous iteration of this loop may have in turn generated
		// more wakeups. We need to process this before processing any new ops,
		// because new op arrival times are non-deterministic.
		if len(t.wakeups) > 0 {
			// Process any pending wakeups. These may have been generated by the new op,
			// or by an op/goroutine belonging to another ThreadMgr in the system.
			t.processWakeups()
		} else {
			// If there is a new op to process, process it.
			// Get the next op.
			op := t.ops[0]
			t.ops = t.ops[1:]

			// Process the op. Run the next op in a new goroutine (which may sleep or
			// block).
			go func() {
				// Execute the op.
				t.pfn(op.Fc, op.replies)
				// Lock to make sure the completion signal isn't missed.
				t.Lock()
				// Mark the op as no longer executing.
				delete(t.executing, op)
				// Notify the ThreadMgr that the op has completed.
				t.cond.Signal()
				// Unlock to allow the ThreadMgr to make progress.
				t.Unlock()
			}()
			// Wait for the op to sleep or complete.
			t.cond.Wait()
		}
	}
}

// Processes wakeups.
func (t *ThreadMgr) processWakeups() {
	// Create a copy of the list of pending wakeups.
	tmp := make([]*sync.Cond, len(t.wakeups))
	for i, c := range t.wakeups {
		tmp[i] = c
	}
	// Empty the wakeups list.
	t.wakeups = t.wakeups[:0]
	// wake up each goroutine/op, then wait for it to complete or sleep again.
	for _, c := range tmp {
		// Wake up the sleeping goroutine.
		c.Signal()
		// Wait for the operation to sleep or terminate.
		t.cond.Wait()
	}
}

func (t *ThreadMgr) start() {
	go t.run()
}

func (t *ThreadMgr) stop() {
	t.Lock()
	defer t.Unlock()
	t.done = true
	t.cond.Signal()
}
