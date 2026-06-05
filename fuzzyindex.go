package mind

import (
	"iter"
)

const (
	FuzzyIndexName      = "FuzzyIndex"
	defaultFuzzyMaxDist = 2
)

type bkEdge struct {
	node *bkNode
	dist uint8
}

type bkNode struct {
	word     string
	ids      *RawIDs32
	children []bkEdge
	// largest edge distance among children, for search optimizing
	maxChild uint8
}

// FuzzyIndex indexes strings in a BK-tree for Levenshtein-distance fuzzy search.
// Use OpFuzzy in Match for default distance 2, or MatchMany for a custom distance.
type FuzzyIndex[OBJ any] struct {
	handler SingleValueHandler[OBJ, string]
	root    *bkNode
}

func NewFuzzyIndex[OBJ any](fieldGetFn FromField[OBJ, string]) Index[OBJ] {
	return &FuzzyIndex[OBJ]{
		handler: SingleValueHandler[OBJ, string]{fieldGetFn},
	}
}

func (fi *FuzzyIndex[OBJ]) Set(obj *OBJ, lidx uint32) {
	fi.handler.Handle(obj, func(s string) {
		if fi.root == nil {
			fi.root = &bkNode{word: s, ids: NewRawIDsFrom(lidx)}
			return
		}

		node := fi.root
		for {
			dist := levenshtein(node.word, s)
			if dist == 0 {
				node.ids.Set(lidx)
				return
			}

			var child *bkNode
			for i := range node.children {
				if int(node.children[i].dist) == dist {
					child = node.children[i].node
					break
				}
			}

			if child == nil {
				child = &bkNode{word: s, ids: NewRawIDsFrom(lidx)}
				node.children = append(node.children, bkEdge{node: child, dist: uint8(dist)})
				if uint8(dist) > node.maxChild {
					node.maxChild = uint8(dist)
				}
				return
			}
			node = child
		}
	})
}

func (fi *FuzzyIndex[OBJ]) UnSet(obj *OBJ, lidx uint32) {
	fi.handler.Handle(obj, func(s string) {
		node := fi.root
		for node != nil {
			dist := levenshtein(node.word, s)
			if dist == 0 {
				node.ids.UnSet(lidx)
				return
			}

			var nextNode *bkNode
			for i := range node.children {
				if int(node.children[i].dist) == dist {
					nextNode = node.children[i].node
					break
				}
			}
			node = nextNode
		}
	})
}

func (fi *FuzzyIndex[OBJ]) BulkSet(objs iter.Seq2[int, *OBJ]) {
	for i, obj := range objs {
		fi.Set(obj, uint32(i))
	}
}

func (fi *FuzzyIndex[OBJ]) HasChanged(oldItem, newItem *OBJ) bool {
	return fi.handler.HasChanged(oldItem, newItem)
}

func (fi *FuzzyIndex[OBJ]) Equal(value any) (*RawIDs32, error) {
	return nil, InvalidOperationError{FuzzyIndexName, OpEq}
}

func (fi *FuzzyIndex[OBJ]) Match(_ *RawIDs32, op FilterOp, value any) (*RawIDs32, bool, error) {
	if op.Op != OpFuzzy {
		return nil, false, InvalidOperationError{FuzzyIndexName, op.Op}
	}

	s, err := ValueFromAny[string](value)
	if err != nil {
		return nil, false, InvalidValueTypeError[string]{value}
	}
	return bkSearch(fi.root, s, defaultFuzzyMaxDist), true, nil
}

func (fi *FuzzyIndex[OBJ]) MatchMany(op FilterOp, values ...any) (*RawIDs32, bool, error) {
	if op.Op != OpFuzzy {
		return nil, false, InvalidOperationError{FuzzyIndexName, op.Op}
	}

	if len(values) != 2 {
		return nil, false, InvalidArgsLenError{Defined: "2", Got: len(values)}
	}
	s, err := ValueFromAny[string](values[0])
	if err != nil {
		return nil, false, err
	}
	dist, err := ValueFromAny[int64](values[1])
	if err != nil {
		return nil, false, err
	}
	return bkSearch(fi.root, s, int(dist)), true, nil
}

// bkSearch walks the BK-tree collecting all words within maxDist of query.
// Iterative stack-based traversal to avoid deep recursion on large trees.
func bkSearch(root *bkNode, query string, maxDist int) *RawIDs32 {
	result := NewRawIDs[uint32]()
	if root == nil {
		return result
	}

	stack := make([]*bkNode, 1, 64)
	stack[0] = root

	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// we only need the exact distance up to maxChild+maxDist: beyond that the
		// node is not a hit and no child edge can fall in [dist-maxDist, dist+maxDist],
		// so the whole subtree is unreachable. Leaves (maxChild==0) cut off at maxDist.
		bound := int(node.maxChild) + maxDist
		dist := levenshteinMax(node.word, query, bound)
		if dist > bound {
			continue
		}
		if dist <= maxDist {
			result.Or(node.ids)
		}

		// only visit children whose edge distance falls in [dist-maxDist, dist+maxDist].
		// This is the core BK-tree pruning property.
		lo := max(dist-maxDist, 0)
		hi := dist + maxDist

		for i := range node.children {
			d := int(node.children[i].dist)
			if d >= lo && d <= hi {
				stack = append(stack, node.children[i].node)
			}
		}
	}

	return result
}

// levenshtein returns the Levenshtein edit distance between a and b.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	if a == b {
		return 0
	}

	// keep a as the shorter string to minimise allocations
	if la > lb {
		a, b = b, a
		la, lb = lb, la
	}

	var stack [32]int
	var prev, curr []int

	if la+1 <= 16 {
		prev = stack[:la+1]
		curr = stack[16 : 16+la+1]
	} else {
		// safe fallback for unusually long string anomalies
		prev = make([]int, la+1)
		curr = make([]int, la+1)
	}

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= lb; i++ {
		curr[0] = i
		for j := 1; j <= la; j++ {
			cost := 1
			if b[i-1] == a[j-1] {
				cost = 0
			}
			v := prev[j] + 1
			if ins := curr[j-1] + 1; ins < v {
				v = ins
			}
			if sub := prev[j-1] + cost; sub < v {
				v = sub
			}
			curr[j] = v
		}
		prev, curr = curr, prev
	}

	return prev[la]
}

// levenshteinMax returns the Levenshtein distance between a and b when it is at
// most k, and otherwise some value > k as soon as the bound is provably exceeded.
// Two cheap guards keep far-apart words from paying the full O(la*lb) matrix:
// the length difference is a lower bound on the distance, and the per-row minimum
// is non-decreasing, so once it passes k the final distance cannot come back down.
func levenshteinMax(a, b string, k int) int {
	la, lb := len(a), len(b)

	// keep a as the shorter string to minimise allocations
	if la > lb {
		a, b = b, a
		la, lb = lb, la
	}

	// edit distance is at least the difference in length
	if lb-la > k {
		return k + 1
	}
	if la == 0 {
		return lb
	}
	if a == b {
		return 0
	}

	var stack [32]int
	var prev, curr []int

	if la+1 <= 16 {
		prev = stack[:la+1]
		curr = stack[16 : 16+la+1]
	} else {
		// Safe fallback for unusually long string anomalies
		prev = make([]int, la+1)
		curr = make([]int, la+1)
	}

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= lb; i++ {
		curr[0] = i
		rowMin := i
		for j := 1; j <= la; j++ {
			cost := 1
			if b[i-1] == a[j-1] {
				cost = 0
			}
			v := prev[j] + 1
			if ins := curr[j-1] + 1; ins < v {
				v = ins
			}
			if sub := prev[j-1] + cost; sub < v {
				v = sub
			}
			curr[j] = v
			if v < rowMin {
				rowMin = v
			}
		}
		if rowMin > k {
			return k + 1
		}
		prev, curr = curr, prev
	}

	return prev[la]
}
