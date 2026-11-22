// Package namelen flags declared names too long to read at a glance.
//
// A name with more characters than its limit is reported:
//   - MaxLocalNameLen (20) for a local variable, parameter or named result,
//     declared with :=, var or range, in a function or a func literal;
//   - MaxTestNameLen (50) for a go test Test, Benchmark, Fuzz or Example
//     function, in any file, since shared test suites keep such functions
//     outside _test.go files;
//   - MaxNameLen (30) for every other function, method, interface method, type,
//     struct field, constant and package-level variable.
//
// Receivers, type parameters, labels, import names and parameter names in
// function types and interface methods are not checked.
// A //nolint:namelen comment on the declaring line silences a report.
package namelen
