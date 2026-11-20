// Package funcorder orders a file's functions by control flow, callers first.
//
// Within a file:
//   - exported functions sit above unexported ones, as do main and init;
//   - a function sits below the first function in the file that calls or
//     references it, so helpers read top to bottom after their callers;
//   - a type's methods sit together below the type, ordered by the same rules,
//     and an unexported type's methods sit below the exported functions.
//
// Calls are matched by name, and methods only through their receiver, so
// calls across files or through other values are not constrained. Recursion
// and call cycles are skipped, and an exported function is only held below
// exported callers. //nolint:funcorder on a function's doc lifts the caller
// and exported-first rules; an exempt method still stays with its type.
// Fix reorders a file so that every rule holds, moving each declaration with
// the comments above it.
package funcorder
