package helpers

import (
	"container/heap"

	"github.com/shayunak/SatSimGo/connections"
)

type Item struct {
	Value *connections.Event
	Rank  int
	Index int
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) IsEmpty() bool { return len(pq) == 0 }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Rank < pq[j].Rank
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) update(item *Item, event *connections.Event, rank int) {
	item.Value = event
	item.Rank = rank
	heap.Fix(pq, item.Index)
}
