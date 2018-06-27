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

package core

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

func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		items:      make([]*node, 10),
		keyToIndex: map[string]int{},
		size:       0,
	}
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

// Adds an element.  'key' must be unique.
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

// Pop removes the highest priority element, returning an error if empty.
func (pq *PriorityQueue) Pop() (interface{}, error) {
	if pq.size == 0 {
		return nil, fmt.Errorf("cannot pop -- priority queue empty")
	}
	item := pq.items[0]
	pq.size--
	last := pq.items[pq.size]
	pq.items[0] = last
	pq.keyToIndex[last.key] = 0
	pq.items[pq.size] = nil
	pq.siftDown(0)
	return item.value, nil
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
	pq.keyToIndex[pq.items[i].key] = j
	pq.keyToIndex[pq.items[j].key] = i
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
