// Package db provides context-based database connection management,
// storing connections and transactions in [context.Context] values
// for seamless function composition across CRUD operations.
//
// # Security model
//
// This package forwards raw SQL fragments (query, whereCondition,
// returningColumns, conflictTarget) from its exported functions directly
// into the generated SQL without parameterization or validation. Those
// parameters MUST be static SQL written by the developer and MUST NOT
// contain data originating from external input. whereCondition,
// returningColumns and conflictTarget are inserted by the builder
// surrounded by the appropriate keyword (WHERE, RETURNING, ON CONFLICT /
// ON DUPLICATE KEY UPDATE / MERGE depending on the driver), so they must
// NOT include those keywords themselves. Pass external input through the
// variadic args slice using the driver's placeholder syntax instead.
//
// See the security model section of the parent sqldb package documentation
// for the full list of raw-fragment parameters and concrete examples.
package db
