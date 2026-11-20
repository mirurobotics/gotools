// Package funcorder orders a file's functions by control flow, callers first.
//
// Within a file:
//   - a function sits directly below its last caller, exported or not, after
//     the functions that caller calls first and their helpers, so a file reads
//     depth-first in call order and has one order. A type's methods count as
//     one caller; a const or var block between functions moves below the
//     calls it splits;
//   - an exported entry point (a function nothing in the file calls), main or
//     init sits above unexported functions that nothing in the file calls;
//   - a type's methods sit together below the type. A type declared on its own
//     is followed by the const and var blocks directly below it, then its
//     methods; Fix leaves behind blocks that would reorder initialized vars;
//   - a type declared on its own without methods sits directly above its first
//     user, a function or var block, with the const blocks below it using it.
//
// Calls are matched by name, methods only through their receiver. Calls within
// a cycle don't order its members. //nolint:funcorder on a function's doc
// lifts the caller, call order and entry point rules. Fix moves each
// declaration with its comments.
package funcorder
