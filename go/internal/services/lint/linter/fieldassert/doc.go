// Package fieldassert flags tests that check a struct one field at a time.
//
// A block of testify equality asserts (Equal, EqualValues, Exactly,
// EqualExportedValues and their f variants) that compare fields of one value,
//
//	assert.Equal(t, 1, got.A)
//	assert.Equal(t, "x", got.B)
//	assert.Equal(t, true, got.C)
//
// is reported once it covers MinFieldAsserts (3) distinct fields. A statement
// that may change the value starts a new group: any use outside a testify call,
// or a closure, & or call on it inside one, except conversions to predeclared
// types and the pure builtins len, cap, min, max, real, imag and complex. Two
// field checks are often targeted; from three on the test rebuilds the struct.
//
// The actual argument decides the group, so table tests comparing tc.wantA to
// got.A are grouped by got; the expected one counts only when the actual is a
// literal or package constant (swapped arguments). Asserts in a dedicated check
// helper, a function that takes the value as a parameter or receiver, are
// allowed, including in closures inside it such as t.Run.
package fieldassert
