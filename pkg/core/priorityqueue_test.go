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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type keyVal struct {
	key      string
	priority int
	value    string
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
		})
		It("should return an error when popping", func() {
			next, err := pq.Pop()
			Expect(next).Should(BeNil())
			Expect(err).ShouldNot(BeNil())
		})
		It("isEmpty", func() {
			Expect(pq.IsEmpty()).To(Equal(true))
		})
	})

	// add
	Describe("Add and Pop", func() {
		pq := NewPriorityQueue()
		It("should allow us to add several elements", func() {
			for _, s := range fiveKeyVals {
				err := pq.Add(s.key, s.priority, s.value)
				Expect(err).To(BeNil())
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
		})
		It("should produce the elements in sorted order, regardless of insertion order", func() {
			expected := []keyVal{four, two, three, five, one}
			for i := 0; !pq.IsEmpty(); i++ {
				elem, err := pq.Pop()
				Expect(elem).ToNot(BeNil())
				Expect(elem).To(Equal(expected[i].value))
				Expect(err).To(BeNil())
			}
		})
	})

	// remove

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
})
