// Package varname flags variables whose names are too short to read.
//
// A variable, parameter or named result declared with a name of fewer than
// MinNameLen (3) characters is reported, in tests as in other code:
//
//	n := len(items)   // reported: use count, size, ...
//	for k, v := range m { // both reported: use key, val, ...
//
// The names _, i, a, b, in, ok, id, db, wg and mu are allowed everywhere,
// whatever their type: i is meant for iteration, a and b for comparison, in for
// an input, ok for the comma-ok idiom, and id, db, wg and mu are common Go
// shorthands. In _test.go files tc is allowed too, for a test case in
// table-driven tests. Receivers are left to rcvrname, which asks for 1-2 chars,
// and type parameters keep Go's single-letter convention. Parameters may take
// the conventional name of c *gin.Context, tx *gorm.DB, w http.ResponseWriter
// and r *http.Request, and any name for a testing type (*testing.T, B, F, M,
// PB, testing.TB) or a TestingI interface, which assert helpers take instead.
// Constants, struct fields and parameter names in function types and
// interfaces are not checked.
// A //nolint:varname comment on the declaring line silences a report.
package varname
