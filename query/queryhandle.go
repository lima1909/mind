package query

import (
	"github.com/lima1909/mind/lidx"
)

type Opion func(*queryOptions)

type queryOptions struct {
	withOptimizer bool
	withTracer    *Tracer
}

func newDefaultQueryOption() queryOptions { return queryOptions{withOptimizer: true} }
func NoOptimizer() Opion                  { return func(o *queryOptions) { o.withOptimizer = false } }
func WithTracer(t *Tracer) Opion          { return func(o *queryOptions) { o.withTracer = t } }

type HandleFNs[T any] struct {
	ReadQuery  func(Query, func(*lidx.RawIDs32)) error
	WriteQuery func(Query, func(*lidx.RawIDs32)) error
	GetItem    func(int) (T, bool)
	RemoveItem func(int) bool
	UpdateItem func(index int, update func(*T)) error
}

type QHandle[T any] struct {
	query Query
	fns   HandleFNs[T]
	err   error
}

func NewQHandleFromStr[T any](handleFNs HandleFNs[T], queryStr string, opts ...Opion) QHandle[T] {
	ast, err := Parse(queryStr)
	if err != nil {
		var query Query
		return QHandle[T]{query: query, fns: handleFNs, err: err}
	}

	return NewQHandleFromExpr(handleFNs, ast, opts...)
}

func NewQHandleFromExpr[T any](handleFNs HandleFNs[T], query Expr, opts ...Opion) QHandle[T] {
	opt := newDefaultQueryOption()
	for _, o := range opts {
		o(&opt)
	}

	if opt.withOptimizer {
		query = query.Optimize()
	}

	q := query.Compile(opt.withTracer)
	return QHandle[T]{query: q, fns: handleFNs}
}

// QHandle a handle for executing queries which have NoIDs
func (h QHandle[T]) Count() (int, error) {
	count := 0
	if h.err != nil {
		return count, h.err
	}

	return count, h.fns.ReadQuery(h.query, func(rids *lidx.RawIDs32) {
		count = rids.Count()
	})
}

func (h QHandle[T]) Values() ([]T, error) {
	var result []T
	if h.err != nil {
		return result, h.err
	}

	return result, h.fns.ReadQuery(h.query, func(rids *lidx.RawIDs32) {
		result = make([]T, 0, rids.Count())

		rids.Values(func(idx uint32) bool {
			item, _ := h.fns.GetItem(int(idx))
			result = append(result, item)

			return true
		})
	})
}

func (h QHandle[T]) Remove() (int, error) {
	if h.err != nil {
		return 0, h.err
	}

	count := 0
	return count, h.fns.WriteQuery(h.query, func(rids *lidx.RawIDs32) {
		rids.Values(func(idx uint32) bool {
			if h.fns.RemoveItem(int(idx)) {
				count++
			}
			return true
		})
	})
}

func (h QHandle[T]) Update(update func(*T)) (int, error) {
	if h.err != nil {
		return 0, h.err
	}

	count := 0
	var errr error
	err := h.fns.WriteQuery(h.query, func(rids *lidx.RawIDs32) {
		rids.Values(func(idx uint32) bool {

			errr = h.fns.UpdateItem(int(idx), update)
			if errr != nil {
				return false
			}

			count++
			return true
		})
	})

	if errr != nil {
		return 0, errr
	}

	return count, err
}

// Paginate the result values of the Query, but in Pagination
func (h QHandle[T]) Paginate(offset, limit uint32) ([]T, PageInfo, error) {
	var result []T
	var pi PageInfo

	if h.err != nil {
		return result, pi, h.err
	}

	return result, pi, h.fns.ReadQuery(h.query, func(rids *lidx.RawIDs32) {
		total := uint32(rids.Count())
		pi = Paginate{offset, limit}.computePageInfo(total)
		result = make([]T, 0, rids.Count())

		rids.ValuesSkipN(int(pi.Offset), func(idx uint32) bool {
			item, _ := h.fns.GetItem(int(idx))
			result = append(result, item)

			// run only until reach the limit
			return uint32(len(result)) < pi.Limit
		})
	})
}

type Paginate struct {
	Offset uint32
	Limit  uint32
}

type PageInfo struct {
	Offset uint32
	Limit  uint32
	Count  uint32
	Total  uint32
}

func (p Paginate) computePageInfo(total uint32) PageInfo {
	offset := p.Offset
	limit := total // default to "all"
	// if limit is provided and not zero, use it; otherwise stay at "total"
	if p.Limit > 0 {
		limit = p.Limit
	}
	pi := PageInfo{Offset: offset, Limit: limit, Total: total}

	// bound check
	if offset >= total {
		return pi
	}

	// adjust limit if it exceeds the remaining items
	if offset+limit > total {
		limit = total - offset
	}

	// pi.Limit = limit // overwrite the given Limit with the "real" limit?
	pi.Count = limit
	return pi
}
