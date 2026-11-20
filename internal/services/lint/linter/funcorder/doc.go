// Package funcorder orders a file's functions by control flow, callers first.
//
// Within a file:
//   - a function sits below every function in the file that calls or
//     references it, exported or not, so a file reads top to bottom from its
//     entry points down through their helpers;
//   - an exported entry point (a function nothing in the file calls), main or
//     init sits above unexported functions that nothing in the file calls;
//   - a type's methods sit together below the type, ordered by the same rules;
//     a type declared on its own sits directly above its methods, after any
//     const and var blocks that follow it up to the next type, and Fix moves
//     them together, leaving an initialized var in place if it would pass
//     another, so package initialization order is kept.
//
// Calls are matched by name, and methods only through their receiver, so
// calls across files or through other values are not constrained. Recursion
// and call cycles are skipped. //nolint:funcorder on a function's doc lifts
// the caller and entry point rules; an exempt method still stays with its
// type. Fix moves each declaration with its comments and places a moved
// helper right after its last caller.
package funcorder
