// Package fieldassert flags tests that check a struct one field at a time.
//
// A block of testify equality asserts (Equal, EqualValues, Exactly,
// EqualExportedValues and their f variants) that compare fields of one value,
//
//	assert.Equal(t, 1, got.A)
//	assert.Equal(t, "x", got.B)
//	assert.Equal(t, true, got.C)
//
// is reported once it covers MinFieldAsserts (3) distinct fields. Assigning to
// or incrementing the value or one of its fields starts a new group. Two field
// checks are often targeted (an ID and a status); from three on the test
// rebuilds the struct by hand and silently stops covering it when a field is
// added. Compare the whole value instead.
//
// The actual argument decides the group, so table tests comparing tc.wantA to
// got.A are grouped by got; the expected one counts only when the actual is a
// literal or package constant (swapped arguments). Asserts in a dedicated check
// helper, a function that takes the value as a parameter or receiver, are
// allowed, including in closures inside it such as t.Run.
package fieldassert
