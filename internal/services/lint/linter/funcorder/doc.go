// Package funcorder orders a file's functions by control flow, callers first.
//
// Within a file:
//   - a function sits below every function in the file that calls or
//     references it, exported or not, so a file reads top to bottom from its
//     entry points down through their helpers;
//   - an exported entry point (a function nothing in the file calls), main or
//     init sits above unexported functions that nothing in the file calls;
//   - a type's methods sit together below the type. A type declared on its own
//     sits directly above its methods, after the const and var blocks below
//     it, and Fix moves them together, leaving the blocks from the first
//     initialized var on if moving them would reorder initialized vars;
//   - a type without methods sits directly above its first user, a function
//     or const or var block; a type with methods and its blocks count as one.
//
// Calls are matched by name and methods only through their receiver. Call
// cycles are skipped. //nolint:funcorder on a function's doc lifts the caller
// and entry point rules. Fix moves each declaration with its comments.
package funcorder
