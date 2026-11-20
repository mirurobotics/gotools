// Package fieldassert flags tests that check a struct one field at a time.
//
// A block of testify equality asserts (Equal, EqualValues, Exactly,
// EqualExportedValues and their f variants) that compare fields of one value,
//
//	assert.Equal(t, 1, got.A)
//	assert.Equal(t, "x", got.B)
//	assert.Equal(t, true, got.C)
//
// is reported once it covers MinFieldAsserts (3) distinct fields; reassigning
// the value starts a new group. Two field checks are often targeted (an ID and
// a status); from three on the test rebuilds the struct by hand and silently
// stops covering it when a field is added. Compare the whole value instead.
//
// The actual argument decides the group, so table tests comparing tc.wantA to
// got.A are grouped by got; the expected one counts only when the actual is a
// literal (swapped arguments). Asserts inside a function or
// closure that takes the checked value as a parameter are allowed: that is a
// dedicated check helper. A method's receiver and the parameters of enclosing
// functions count too, so t.Run closures inside a helper are allowed.
package fieldassert
