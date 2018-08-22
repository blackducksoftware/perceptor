/*
Copyright (C) 2018 Synopsys, Inc.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements. See the NOTICE file
distributed with this work for additional information
regarding copyright ownership. The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the License for the
specific language governing permissions and limitations
under the License.
*/

package util

import (
	"encoding/json"
	"fmt"
)

type node struct {
	key      string
	priority int
	value    interface{}
}

// PriorityQueue uses a max heap, and provides efficient changing of priority.
type PriorityQueue struct {
	items      []*node
	size       int
	keyToIndex map[string]int
}

// NewPriorityQueue .....
func NewPriorityQueue() *PriorityQueue {
	return newPriorityQueueWithInitialCapacity(10)
}

func newPriorityQueueWithInitialCapacity(capacity int) *PriorityQueue {
	return &PriorityQueue{
		items:      make([]*node, capacity),
		keyToIndex: map[string]int{},
		size:       0,
	}
}

// Values should only be used for debugging.
func (pq *PriorityQueue) Values() []interface{} {
	elems := make([]interface{}, pq.size)
	for i := 0; i < pq.size; i++ {
		elems[i] = pq.items[i].value
	}
	return elems
}

// Dump should only be used for debugging.
func (pq *PriorityQueue) Dump() []map[string]interface{} {
	elems := make([]map[string]interface{}, pq.size)
	for i := 0; i < pq.size; i++ {
		elems[i] = map[string]interface{}{
			"Key":      pq.items[i].key,
			"Priority": pq.items[i].priority,
		}
	}
	return elems
}

// DebugString should only be used for debugging.
func (pq *PriorityQueue) DebugString() string {
	items := make([]map[string]interface{}, len(pq.items))
	for i, item := range pq.items {
		if i >= pq.size {
			break
		}
		items[i] = map[string]interface{}{
			"key":      item.key,
			"priority": item.priority,
			"value":    item.value,
		}
	}
	dict := map[string]interface{}{
		"keyToIndex": pq.keyToIndex,
		"size":       pq.size,
		"items":      items,
	}
	jsonBytes, err := json.Marshal(dict)
	if err != nil {
		return err.Error()
	}
	return string(jsonBytes)
}

// Add adds an element.  'key' must be unique.
func (pq *PriorityQueue) Add(key string, priority int, value interface{}) error {
	if _, ok := pq.keyToIndex[key]; ok {
		return fmt.Errorf("cannot add key %s: key already in map", key)
	}
	pq.resizeIfNecessary()
	pq.items[pq.size] = &node{key: key, priority: priority, value: value}
	pq.keyToIndex[key] = pq.size
	pq.siftUp(pq.size)
	pq.size++
	return nil
}

// Peek returns the highest priority item, or nil if empty.
func (pq *PriorityQueue) Peek() interface{} {
	if pq.size == 0 {
		return nil
	}
	return pq.items[0].value
}

// Pop removes the highest priority element, returning an error if empty.
func (pq *PriorityQueue) Pop() (interface{}, error) {
	if pq.size == 0 {
		return nil, fmt.Errorf("cannot pop -- priority queue empty")
	}
	item := pq.items[0]
	// clean up
	delete(pq.keyToIndex, item.key)
	pq.size--
	last := pq.items[pq.size]
	pq.items[pq.size] = nil
	// restore heap property
	if pq.size > 0 {
		pq.items[0] = last
		pq.keyToIndex[last.key] = 0
		pq.siftDown(0)
	}
	// done
	return item.value, nil
}

// Size returns the number of elements in the queue.
func (pq *PriorityQueue) Size() int {
	return pq.size
}

// IsEmpty .....
func (pq *PriorityQueue) IsEmpty() bool {
	return pq.size == 0
}

// Set changes the priority of 'value' if it can be found, and returns an error if not.
func (pq *PriorityQueue) Set(key string, priority int) error {
	index, ok := pq.keyToIndex[key]
	if !ok {
		return fmt.Errorf("cannot change priority of key %s, key not found", key)
	}
	node := pq.items[index]
	node.priority = priority
	pq.siftUp(index)
	pq.siftDown(index)
	return nil
}

// HasKey returns whether the priority queue has the key.
func (pq *PriorityQueue) HasKey(key string) bool {
	_, ok := pq.keyToIndex[key]
	return ok
}

// Remove removes the value associated with the key from the priority queue,
// returning an error if it can't be found.
func (pq *PriorityQueue) Remove(key string) (interface{}, error) {
	index, ok := pq.keyToIndex[key]
	if !ok {
		return nil, fmt.Errorf("cannot remove key %s, key is not present", key)
	}
	item := pq.items[index]
	// if it's not the last one: must restore the heap property
	// example: remove index 6, initial size was 7 => don't restore
	// example: remove index 5, initial size was 7 => swap index 6 into 5, restore
	lastIndex := pq.size - 1
	if index != lastIndex {
		pq.items[index] = pq.items[lastIndex]
		pq.keyToIndex[pq.items[index].key] = index
		pq.items[lastIndex] = nil
		pq.size--
		pq.siftUp(index)
		pq.siftDown(index)
	} else {
		pq.items[lastIndex] = nil
		pq.size--
	}
	delete(pq.keyToIndex, key)
	return item.value, nil
}

// CheckValidity should always return an empty slice -- it is just a debugging tool.
// If it returns a non-empty slice, then there's a bug somewhere in the heap code.
func (pq *PriorityQueue) CheckValidity() []string {
	errors := []string{}
	// check the heap property
	for i := 0; i < pq.size; i++ {
		lc := leftChild(i)
		if lc >= pq.size {
			break
		}
		curr := pq.items[i]
		left := pq.items[lc]
		if left.priority > curr.priority {
			errors = append(errors, fmt.Sprintf("parent %d(%d) has lower priority than left child %d(%d)", i, curr.priority, lc, left.priority))
		}
		rc := rightChild(i)
		if rc >= pq.size {
			break
		}
		right := pq.items[rc]
		if right.priority > curr.priority {
			errors = append(errors, fmt.Sprintf("parent %d(%d) has lower priority than right child %d(%d)", i, curr.priority, rc, right.priority))
		}
	}
	// check that the keyToIndex map is correct
	for key, ix := range pq.keyToIndex {
		if pq.items[ix].key != key {
			errors = append(errors, fmt.Sprintf("key %s, index %d does not match node: %+v", key, ix, pq.items[ix]))
		}
	}
	// check that the keyToIndex map is complete
	if len(pq.keyToIndex) != pq.size {
		errors = append(errors, fmt.Sprintf("keyToIndex size %d but pq.size is %d", len(pq.keyToIndex), pq.size))
	}
	// done
	return errors
}

// Implementation details:

func (pq *PriorityQueue) resizeIfNecessary() {
	if pq.size < len(pq.items) {
		return
	}
	newItems := make([]*node, len(pq.items)*2)
	for ix, val := range pq.items {
		newItems[ix] = val
	}
	pq.items = newItems
}

func (pq *PriorityQueue) swap(i int, j int) {
	temp := pq.items[i]
	pq.items[i] = pq.items[j]
	pq.items[j] = temp
	pq.keyToIndex[pq.items[i].key] = i
	pq.keyToIndex[pq.items[j].key] = j
}

func (pq *PriorityQueue) siftDown(index int) {
	for ip := index; ; {
		inext := ip
		ilc := leftChild(ip)
		if ilc >= pq.size {
			break
		}
		p := pq.items[ip]
		lc := pq.items[ilc]
		if p.priority < lc.priority {
			inext = ilc
		}

		irc := rightChild(ip)
		if irc < pq.size {
			rc := pq.items[irc]
			if pq.items[inext].priority < rc.priority {
				inext = irc
			}
		}
		if inext == ip {
			break
		}
		pq.swap(ip, inext)
		ip = inext
	}
}

func (pq *PriorityQueue) siftUp(index int) {
	for ic := index; ; {
		ip := parent(ic)
		if ip < 0 {
			break
		}
		p := pq.items[ip]
		c := pq.items[ic]
		if c.priority > p.priority {
			pq.swap(ic, ip)
		}
		if ic <= 0 {
			break
		}
		ic = ip
	}
}

func parent(index int) int {
	return (index - 1) / 2
}

func leftChild(index int) int {
	return index*2 + 1
}

func rightChild(index int) int {
	return index*2 + 2
}
