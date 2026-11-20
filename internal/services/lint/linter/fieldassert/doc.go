// Package fieldassert flags tests that check a struct one field at a time.
//
// A block of testify equality asserts (Equal, EqualValues, Exactly and their
// f variants) that each compare a field of the same value, such as
//
//	assert.Equal(t, 1, got.A)
//	assert.Equal(t, "x", got.B)
//	assert.Equal(t, true, got.C)
//
// is reported once it reaches MinFieldAsserts (3). Two field checks are often
// a targeted assertion, such as an ID and a status; from three on the test is
// rebuilding the struct by hand, and it silently stops covering the struct
// when a field is added. Compare the whole value in one assert instead.
//
// The actual argument decides the group, so table tests comparing tc.wantA to
// got.A are grouped by got; the expected one counts only when the actual is a
// literal (swapped arguments). Asserts inside a function or
// closure that takes the checked value as a parameter are allowed: that is a
// dedicated check helper.
package fieldassert
