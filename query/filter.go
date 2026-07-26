package query

import "github.com/lima1909/mind/lidx"

var (
	FOpEq      = FilterOp{Op: OpEq}
	FOpNeq     = FilterOp{Op: OpNeq}
	FOpLe      = FilterOp{Op: OpLe}
	FOpLt      = FilterOp{Op: OpLt}
	FOpGe      = FilterOp{Op: OpGe}
	FOpGt      = FilterOp{Op: OpGt}
	FOpLike    = FilterOp{Op: OpLike}
	FOpSounds  = FilterOp{Op: OpSounds}
	FOpFuzzy   = FilterOp{Op: OpFuzzy}
	FOpIn      = FilterOp{Op: OpIn}
	FOpBetween = FilterOp{Op: OpBetween}
)

// FilterOp is a wrapper over the Op, which contains the Op and a String.
// For User defined FilterOp is no Op defined, so the User defined Index can use the String.
type FilterOp struct {
	Op   Op
	Name string
}

func (f FilterOp) String() string {
	if f.Name != "" {
		return f.Name
	}
	return f.Op.String()

}

// Filter returns the RawIDs or an error by a given Relation and Value
type Filter interface {
	// Equal is seperated from Match
	// because the RawIDs result you can NOT mutable
	Equal(value any) (*lidx.RawIDs32, error)
	// Match execute a query (FilterOP) with one given value
	// for example: age > 18
	Match(allIDs *lidx.RawIDs32, op FilterOp, value any) (ids *lidx.RawIDs32, canMutate bool, err error)
	// MatchMany execute a query (FilterOp) for many given values
	// for example: age between 18 and 80
	MatchMany(op FilterOp, values ...any) (ids *lidx.RawIDs32, canMutate bool, err error)
}
