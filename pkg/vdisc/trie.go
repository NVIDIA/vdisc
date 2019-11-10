// Copyright © 2019 NVIDIA Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vdisc

import (
	"unicode/utf8"

	"github.com/badgerodon/collections/queue"

	"github.com/NVIDIA/vdisc/pkg/iso9660"
)

type InvertedTrieNode struct {
	Parent  int
	Content string
}

type trieNode struct {
	content  string
	terminal bool
	value    iso9660.LogicalBlockAddress
	children map[rune]*trieNode
}

type TrieMap struct {
	root *trieNode
}

func NewTrieMap() *TrieMap {
	return &TrieMap{
		root: &trieNode{
			content:  "",
			terminal: false,
			children: make(map[rune]*trieNode),
		},
	}
}

func (t *TrieMap) Put(key string, value iso9660.LogicalBlockAddress) {
	node := t.root
	contentIdx := 0

	for _, r := range key {
		if contentIdx < len(node.content) {
			cr, crLen := utf8.DecodeRuneInString(node.content[contentIdx:])
			if r == cr {
				contentIdx += crLen
			} else {
				// split the current node
				other := &trieNode{
					content:  node.content[contentIdx:],
					terminal: node.terminal,
					value:    node.value,
					children: node.children,
				}

				node.content = node.content[:contentIdx]
				node.children = make(map[rune]*trieNode)
				node.terminal = false
				node.children[cr] = other

				newNode := &trieNode{
					content:  string(r),
					terminal: false,
					children: make(map[rune]*trieNode),
				}

				node.children[r] = newNode
				node = newNode
				contentIdx = utf8.RuneLen(r)
			}
		} else {
			child, ok := node.children[r]
			if ok {
				node = child
				contentIdx = utf8.RuneLen(r)
			} else if node.terminal || len(node.children) > 0 {
				// forced to split to preserve terminal
				newNode := &trieNode{
					content:  string(r),
					terminal: false,
					children: make(map[rune]*trieNode),
				}

				node.children[r] = newNode
				node = newNode
				contentIdx = utf8.RuneLen(r)
			} else {
				// extentd the current node
				node.content = node.content + string(r)
				contentIdx += utf8.RuneLen(r)
			}
		}
	}

	remainder := node.content[contentIdx:]
	if len(remainder) > 0 {
		// we need to split the node
		cr, _ := utf8.DecodeRuneInString(remainder)
		other := &trieNode{
			content:  remainder,
			terminal: node.terminal,
			value:    node.value,
			children: node.children,
		}

		node.content = node.content[:contentIdx]
		node.children = make(map[rune]*trieNode)
		node.children[cr] = other
	}

	node.terminal = true
	node.value = value
}

func (t *TrieMap) Get(key string) (iso9660.LogicalBlockAddress, bool) {
	node := t.root
	contentIdx := 0
	for _, r := range key {
		if contentIdx < len(node.content) {
			cr, crLen := utf8.DecodeRuneInString(node.content[contentIdx:])
			if r == cr {
				contentIdx += crLen
			} else {
				// character doesn't match
				return 0, false
			}
		} else {
			child, ok := node.children[r]
			if ok {
				node = child
				contentIdx = utf8.RuneLen(r)
			} else {
				return 0, false
			}
		}
	}

	if node.terminal && len(node.content[contentIdx:]) == 0 {
		return node.value, true
	}
	return 0, false
}

type traversalNode struct {
	prefix      string
	parentIndex int
	node        *trieNode
}

func (t *TrieMap) Invert() ([]InvertedTrieNode, map[iso9660.LogicalBlockAddress]InvertedTrieNode) {
	// level-order traversal building up inverted trie nodes

	var branches []InvertedTrieNode
	leaves := make(map[iso9660.LogicalBlockAddress]InvertedTrieNode)

	// Add a sentinel node acting as a null terminator
	branches = append(branches, InvertedTrieNode{
		Parent:  0,
		Content: "",
	})

	q := queue.New()
	q.Enqueue(&traversalNode{
		prefix:      "",
		parentIndex: 0,
		node:        t.root,
	})

	for q.Len() > 0 {
		tnode := q.Dequeue().(*traversalNode)

		if tnode.node.terminal {
			if _, ok := leaves[tnode.node.value]; ok {
				panic("never")
			}
			leaves[tnode.node.value] = InvertedTrieNode{
				Parent:  tnode.parentIndex,
				Content: tnode.node.content,
			}
		}

		if len(tnode.node.children) > 0 {
			idx := len(branches)

			branches = append(branches, InvertedTrieNode{
				Parent:  tnode.parentIndex,
				Content: tnode.node.content,
			})

			for _, child := range tnode.node.children {
				q.Enqueue(&traversalNode{
					prefix:      tnode.prefix + tnode.node.content,
					parentIndex: idx,
					node:        child,
				})
			}
		}
	}

	return branches, leaves
}
