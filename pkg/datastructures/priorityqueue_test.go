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
	"fmt"
	"math/rand"
	"sort"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

type keyVal struct {
	key      string
	priority int
	value    string
}

type sortable struct {
	items []int
}

func (s sortable) Len() int {
	return len(s.items)
}

func (s sortable) Less(i, j int) bool {
	return s.items[i] > s.items[j]
}

func (s sortable) Swap(i, j int) {
	temp := s.items[i]
	s.items[i] = s.items[j]
	s.items[j] = temp
}

var _ = Describe("Priority queue", func() {
	one := keyVal{
		key:      "k1",
		priority: 1,
		value:    "v1",
	}
	two := keyVal{
		key:      "k2",
		priority: 4,
		value:    "v2",
	}
	three := keyVal{
		key:      "k3",
		priority: 3,
		value:    "v3",
	}
	four := keyVal{
		key:      "k4",
		priority: 5,
		value:    "v4",
	}
	five := keyVal{
		key:      "k5",
		priority: 2,
		value:    "v5",
	}

	fiveKeyVals := []keyVal{one, two, three, four, five}

	Describe("Empty", func() {
		pq := NewPriorityQueue()
		It("should have size 0", func() {
			Expect(pq.size).Should(Equal(0))
			Expect(pq.CheckValidity()).To(Equal([]string{}))
		})
		It("should return an error when popping", func() {
			next, err := pq.Pop()
			Expect(next).Should(BeNil())
			Expect(err).ShouldNot(BeNil())
			Expect(pq.CheckValidity()).To(Equal([]string{}))
		})
		It("isEmpty", func() {
			Expect(pq.IsEmpty()).To(Equal(true))
			Expect(pq.CheckValidity()).To(Equal([]string{}))
		})
	})

	Describe("Clean up", func() {
		pq := newPriorityQueueWithInitialCapacity(5)
		It("should handle popping when backing slice is full, and clean up correctly", func() {
			pq.Add("abc", 1, 14)
			pq.Add("def", 5, 100)
			pq.Add("ghi", 3, 39)
			pq.Add("jkl", 2, 82)
			pq.Add("mno", 4, 107)
			val, err := pq.Pop()
			Expect(err).To(BeNil())
			Expect(val).To(Equal(100))
			Expect(pq.items[4]).To(BeNil())
		})
	})

	// add
	Describe("Add and Pop", func() {
		pq := NewPriorityQueue()
		It("should allow us to add several elements", func() {
			for _, s := range fiveKeyVals {
				err := pq.Add(s.key, s.priority, s.value)
				Expect(err).To(BeNil())
				Expect(pq.CheckValidity()).To(Equal([]string{}))
			}
		})
		It("should have n elements", func() {
			Expect(pq.IsEmpty()).To(Equal(false))
			Expect(pq.size).To(Equal(5))
		})
		It("should not let us add duplicate keys", func() {
			for _, s := range fiveKeyVals {
				err := pq.Add(s.key, s.priority, s.value)
				Expect(err).ToNot(BeNil())
			}
			Expect(pq.IsEmpty()).To(Equal(false))
			Expect(pq.size).To(Equal(5))
			Expect(pq.CheckValidity()).To(Equal([]string{}))
		})
		It("should produce the elements in sorted order, regardless of insertion order", func() {
			expected := []keyVal{four, two, three, five, one}
			for i := 0; !pq.IsEmpty(); i++ {
				elem, err := pq.Pop()
				Expect(elem).ToNot(BeNil())
				Expect(elem).To(Equal(expected[i].value))
				Expect(err).To(BeNil())
				// log.Infof("pq dump after removing %+v: %s", elem, pq.DebugString())
				Expect(pq.CheckValidity()).To(Equal([]string{}))
			}
		})
	})

	Describe("Lots of adds, some deletes, more adds, more deletes", func() {
		It("Should automatically resize the buffer as necessary", func() {
			pq := NewPriorityQueue()
			for i := 0; i < 500; i++ {
				//log.Infof("add: %d %d", pq.size, len(pq.items))
				err := pq.Add(fmt.Sprintf("%d", i), 0, i)
				Expect(err).To(BeNil())
				Expect(pq.CheckValidity()).To(Equal([]string{}))
			}
			Expect(pq.size).To(Equal(500))
			for j := 0; j < 75; j++ {
				elem, err := pq.Pop()
				Expect(elem).NotTo(BeNil())
				Expect(err).To(BeNil())
				Expect(pq.CheckValidity()).To(Equal([]string{}))
			}
			Expect(pq.size).To(Equal(425))
			for i := 0; i < 75; i++ {
				//log.Infof("add: %d %d", pq.size, len(pq.items))
				err := pq.Add(fmt.Sprintf("part2-%d", i), 0, i+1000)
				Expect(err).To(BeNil())
				Expect(pq.CheckValidity()).To(Equal([]string{}))
			}
			Expect(pq.size).To(Equal(500))
			for j := 0; j < 500; j++ {
				elem, err := pq.Pop()
				Expect(elem).NotTo(BeNil())
				Expect(err).To(BeNil())
				Expect(pq.CheckValidity()).To(Equal([]string{}))
			}
			Expect(pq.size).To(Equal(0))
			Expect(pq.CheckValidity()).To(Equal([]string{}))
		})
	})

	// has key
	Describe("Has key and remove key", func() {
		limit := 1000
		randomKeys := make([]string, limit)
		keyVals := map[string]interface{}{}
		pq := NewPriorityQueue()
		It("should say that every key we added is present", func() {
			for i := 0; i < limit; i++ {
				key := fmt.Sprintf("%d", rand.Int())
				priority := rand.Int()
				err := pq.Add(key, priority, i)
				keyVals[key] = i
				Expect(err).To(BeNil())
				Expect(pq.CheckValidity()).To(Equal([]string{}))
				randomKeys[i] = key
			}
			Expect(pq.size).To(Equal(limit))
			for i := 0; i < limit; i++ {
				Expect(pq.HasKey(randomKeys[i])).To(BeTrue())
			}
		})
		It("should allow us to remove every key, and get the values out that we put in", func() {
			for i := 0; i < limit; i++ {
				key := randomKeys[i]
				elem, err := pq.Remove(key)
				Expect(elem).To(Equal(keyVals[key]))
				Expect(pq.HasKey(key)).To(BeFalse())
				Expect(err).To(BeNil())
				Expect(pq.CheckValidity()).To(Equal([]string{}))
			}
			Expect(pq.size).To(Equal(0))
		})
	})

	// remove by key
	Describe("Remove by key", func() {

	})

	// change key
	Describe("Change priority", func() {
		pq := NewPriorityQueue()
		It("should set things up correctly", func() {
			for _, s := range fiveKeyVals {
				err := pq.Add(s.key, s.priority, s.value)
				Expect(err).To(BeNil())
			}
			Expect(pq.size).To(Equal(5))
		})
		It("should let us change priorities", func() {
			err := pq.Set("k3", 18)
			Expect(err).To(BeNil())
		})
		It("should reflect the changed priorities in the Pop order", func() {
			expected := []keyVal{three, four, two, five, one}
			for i := 0; !pq.IsEmpty(); i++ {
				elem, err := pq.Pop()
				Expect(elem).ToNot(BeNil())
				Expect(elem).To(Equal(expected[i].value))
				Expect(err).To(BeNil())
			}
		})
	})

	// profiling?  large scale performance test?
	Describe("scale test", func() {
		limits := []int{}
		maxPower := 16 // 2^16 = 65536
		for i, val := 0, 1; i < maxPower+1; i, val = i+1, val*2 {
			limits = append(limits, val)
		}
		Context("add, change, and remove should generally require log(n) time", func() {
			for _, limit := range limits {
				itemCount := limit // copy to avoid variable capture
				pq := NewPriorityQueue()
				It(fmt.Sprintf("should insert %d items in n*log(n) time", limit), func() {
					start := time.Now()
					for i := 0; i < itemCount; i++ {
						priority := rand.Int() % 100000
						//log.Infof("add: %d", i)
						err := pq.Add(fmt.Sprintf("%d", i), priority, i)
						//log.Infof("pq: %s", pq.DebugString())
						Expect(err).To(BeNil())
					}
					stop := time.Now()
					log.Infof("insertion of %d items: duration %s", itemCount, stop.Sub(start))
					// log.Infof("pq dump: %s", pq.DebugString())
					Expect(pq.CheckValidity()).To(Equal([]string{}))
				})
				It(fmt.Sprintf("should change %d priorities in n*log(n) time", limit), func() {
					changeStart := time.Now()
					for i := 0; i < itemCount; i++ {
						err := pq.Set(fmt.Sprintf("%d", i), rand.Int()%100000)
						Expect(err).To(BeNil())
					}
					log.Infof("change priority of %d items: duration %s", itemCount, time.Now().Sub(changeStart))
					Expect(pq.CheckValidity()).To(Equal([]string{}))
				})
				It(fmt.Sprintf("should remove %d items in n*log(n) time", limit), func() {
					priorities := make([]int, itemCount)
					removeStart := time.Now()
					for i := 0; i < itemCount; i++ {
						priorities[i] = pq.items[0].priority
						elem, err := pq.Pop()
						Expect(elem).NotTo(BeNil())
						Expect(err).To(BeNil())
					}
					log.Infof("removal of %d items: duration %s", itemCount, time.Now().Sub(removeStart))
					// log.Infof("priorities: %+v", priorities)
					Expect(pq.CheckValidity()).To(Equal([]string{}))
					Expect(checkArrayForSortedness(priorities)).To(Equal([][3]int{}))
					Expect(sort.IsSorted(sortable{items: priorities})).To(BeTrue())
				})
			}
		})
	})
})

func checkArrayForSortedness(arr []int) [][3]int {
	problems := [][3]int{}
	for i := 0; i < len(arr)-1; i++ {
		if arr[i] < arr[i+1] {
			problems = append(problems, [3]int{i, arr[i], arr[i+1]})
		}
	}
	return problems
}
