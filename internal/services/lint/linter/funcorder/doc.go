// Package funcorder orders a file's functions by control flow, callers first.
//
// Within a file:
//   - a function sits directly below its last caller, exported or not, after
//     the functions that caller calls first and their helpers, so a file reads
//     depth-first in call order and has one order. A type's methods count as
//     one caller, and a const or var block directly above a free function
//     moves with it;
//   - an exported entry point (a function nothing in the file calls), main or
//     init sits above unexported functions that nothing in the file calls;
//   - a type's methods sit together below the type. A type declared on its own
//     is followed by the const and var blocks directly below it, then its
//     methods; Fix leaves behind blocks that would reorder initialized vars;
//   - a type declared on its own without methods sits directly above its first
//     user, a function or var block, with the const blocks right above it and
//     those below it that use it.
//
// Calls are matched by name, methods only through their receiver. A call within
// a cycle needn't sit above its callee, but the walk follows it. A function's
// //nolint:funcorder lifts the caller, call order and entry point rules.
package funcorder
