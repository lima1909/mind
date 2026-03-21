// Package mind (Multi INDex list) lets you query in-memory collections
// by multiple fields using indexes, just like a database — but without one.
//
// It is particularly well suited where data is read more often than written.
//
// # Trade-offs
//
// ## Advantages
// - Zero dependencies
// - Generic — works with any struct type
// - Fast reads via bitmap-accelerated index intersection
// - SQL-like query language (with optimizer)
//
// ## Disadvantages
// - Higher memory usage: indexes store additional data alongside user data
// - Slower writes: every mutation updates all registered indexes
package mind
