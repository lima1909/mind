package mind

import (
	"github.com/lima1909/mind/lidx"
	"github.com/lima1909/mind/query"
)

type Opion func(*queryOptions)

type queryOptions struct {
	withOptimizer bool
	withTracer    *query.Tracer
}

func newDefaultQueryOption() queryOptions { return queryOptions{withOptimizer: true} }
func NoOptimizer() Opion                  { return func(o *queryOptions) { o.withOptimizer = false } }
func WithTracer(t *query.Tracer) Opion    { return func(o *queryOptions) { o.withTracer = t } }

type getItemFn[T any] func(lidx uint32) (T, bool)
type executorFn[T any] func(query.Query, func(*lidx.RawIDs32, getItemFn[T])) error

type QHandle[T any] struct {
	query query.Query
	exec  executorFn[T]
	err   error
}

func NewQHandleFromStr[T any](queryExec executorFn[T], queryStr string, opts ...Opion) QHandle[T] {
	ast, err := query.Parse(queryStr)
	if err != nil {
		var query query.Query
		return QHandle[T]{exec: queryExec, query: query, err: err}
	}

	return NewQHandleFromExpr(queryExec, ast, opts...)
}

func NewQHandleFromExpr[T any](queryExec executorFn[T], query query.Expr, opts ...Opion) QHandle[T] {
	opt := newDefaultQueryOption()
	for _, o := range opts {
		o(&opt)
	}

	if opt.withOptimizer {
		query = query.Optimize()
	}

	q := query.Compile(opt.withTracer)
	return QHandle[T]{exec: queryExec, query: q}
}

// QHandle a handle for executing queries which have NoIDs
func (h QHandle[T]) Count() (int, error) {
	count := 0
	if h.err != nil {
		return count, h.err
	}

	return count, h.exec(h.query, func(rids *lidx.RawIDs32, _ getItemFn[T]) {
		count = rids.Count()
	})
}

func (h QHandle[T]) Values() ([]T, error) {
	var result []T
	if h.err != nil {
		return result, h.err
	}

	return result, h.exec(h.query, func(rids *lidx.RawIDs32, getItem getItemFn[T]) {
		result = make([]T, 0, rids.Count())

		rids.Values(func(idx uint32) bool {
			item, _ := getItem(idx)
			result = append(result, item)

			return true
		})
	})
}

// Paginate the result values of the Query, but in Pagination
func (h QHandle[T]) Paginate(offset, limit uint32) ([]T, PageInfo, error) {
	var result []T
	var pi PageInfo

	if h.err != nil {
		return result, pi, h.err
	}

	return result, pi, h.exec(h.query, func(rids *lidx.RawIDs32, getItem getItemFn[T]) {
		total := uint32(rids.Count())
		pi = Paginate{offset, limit}.computePageInfo(total)
		result = make([]T, 0, rids.Count())

		rids.ValuesSkipN(int(pi.Offset), func(idx uint32) bool {
			item, _ := getItem(idx)
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
