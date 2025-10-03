package dice

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/sealdice/smallseal/dice/types"
)

type hookRegistry[T any] struct {
	mu    sync.RWMutex
	seq   atomic.Uint64
	items []hookEntry[T]
	index map[types.HookHandle]int
}

type hookEntry[T any] struct {
	id       types.HookHandle
	name     string
	priority int
	handler  T
}

func (r *hookRegistry[T]) register(name string, priority types.HookPriority, handler T) (types.HookHandle, error) {
	if any(handler) == nil {
		return "", errors.New("hook handler must not be nil")
	}

	id := types.HookHandle(fmt.Sprintf("hook-%d", r.seq.Add(1)))
	entry := hookEntry[T]{
		id:       id,
		name:     name,
		priority: int(priority),
		handler:  handler,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.index == nil {
		r.index = make(map[types.HookHandle]int)
	}

	insertAt := len(r.items)
	for i, existing := range r.items {
		if entry.priority > existing.priority {
			insertAt = i
			break
		}
	}

	r.items = append(r.items, hookEntry[T]{})
	copy(r.items[insertAt+1:], r.items[insertAt:])
	r.items[insertAt] = entry
	r.reindexLocked(insertAt)

	return id, nil
}

func (r *hookRegistry[T]) unregister(handle types.HookHandle) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.items) == 0 {
		return false
	}

	idx, ok := r.index[handle]
	if !ok {
		return false
	}

	r.items = append(r.items[:idx], r.items[idx+1:]...)
	delete(r.index, handle)
	r.reindexLocked(idx)
	return true
}

func (r *hookRegistry[T]) snapshot() []hookEntry[T] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.items) == 0 {
		return nil
	}

	out := make([]hookEntry[T], len(r.items))
	copy(out, r.items)
	return out
}

func (r *hookRegistry[T]) reindexLocked(start int) {
	if r.index == nil {
		r.index = make(map[types.HookHandle]int, len(r.items))
	}

	if start < 0 {
		start = 0
	}

	for i := start; i < len(r.items); i++ {
		r.index[r.items[i].id] = i
	}
}
