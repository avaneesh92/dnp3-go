package queue

import (
	"container/heap"
	"sync"
	"time"
)

// Item represents a priority queue item
type Item struct {
	Value    interface{} // The actual task/value
	Priority int         // Priority (higher = more important)
	NextRun  time.Time   // When this item should run
	Index    int         // Index in the heap
}

// PriorityQueue implements a priority queue with time-based scheduling
type PriorityQueue struct {
	items itemHeap
	mu    sync.Mutex
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make(itemHeap, 0),
	}
	heap.Init(&pq.items)
	return pq
}

// Push adds an item to the queue
func (pq *PriorityQueue) Push(value interface{}, priority int, nextRun time.Time) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &Item{
		Value:    value,
		Priority: priority,
		NextRun:  nextRun,
	}
	heap.Push(&pq.items, item)
}

// Pop removes and returns the highest priority item that is ready to run
func (pq *PriorityQueue) Pop() interface{} {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.items.Len() == 0 {
		return nil
	}

	item := heap.Pop(&pq.items).(*Item)
	return item.Value
}

// Peek returns the highest priority item without removing it
func (pq *PriorityQueue) Peek() *Item {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.items.Len() == 0 {
		return nil
	}

	return pq.items[0]
}

// NextReady returns the next ready item (time has passed) without removing it
func (pq *PriorityQueue) NextReady(now time.Time) interface{} {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.items.Len() == 0 {
		return nil
	}

	// Check if top item is ready
	item := pq.items[0]
	if now.Before(item.NextRun) {
		return nil
	}

	// Remove and return
	item = heap.Pop(&pq.items).(*Item)
	return item.Value
}

// Reschedule reschedules an item
func (pq *PriorityQueue) Reschedule(value interface{}, priority int, nextRun time.Time) {
	pq.Push(value, priority, nextRun)
}

// Len returns the number of items in the queue
func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return pq.items.Len()
}

// Clear removes all items
func (pq *PriorityQueue) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.items = make(itemHeap, 0)
	heap.Init(&pq.items)
}

// itemHeap implements heap.Interface
type itemHeap []*Item

func (h itemHeap) Len() int { return len(h) }

func (h itemHeap) Less(i, j int) bool {
	// First by time (earlier = higher priority)
	if h[i].NextRun.Before(h[j].NextRun) {
		return true
	}
	if h[j].NextRun.Before(h[i].NextRun) {
		return false
	}
	// Then by priority (higher = higher priority)
	return h[i].Priority > h[j].Priority
}

func (h itemHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *itemHeap) Push(x interface{}) {
	item := x.(*Item)
	item.Index = len(*h)
	*h = append(*h, item)
}

func (h *itemHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*h = old[0 : n-1]
	return item
}
