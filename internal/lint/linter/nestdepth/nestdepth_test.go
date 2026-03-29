package nestdepth

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		maxDepth int
		wantN    int
		wantMsg  string
	}{
		{
			name: "at limit",
			src: `package foo

func f() {
	if true {
		for i := 0; i < 10; i++ {
			if true {
				switch {
				case true:
					x := 1
					_ = x
				}
			}
		}
	}
}
`,
			maxDepth: 4,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name: "over limit",
			src: `package foo

func f() {
	if true {
		for i := 0; i < 10; i++ {
			if true {
				switch {
				case true:
					if true {
						x := 1
						_ = x
					}
				}
			}
		}
	}
}
`,
			maxDepth: 4,
			wantN:    1,
			wantMsg: "nesting depth is 5 (limit 4)" +
				" — consider early returns or extracting a helper",
		},
		{
			name: "switch case body no extra depth",
			src: `package foo

func f() {
	switch x := 1; {
	case x > 0:
		if true {
			y := 2
			_ = y
		}
	}
}
`,
			maxDepth: 2,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name: "nested if and for",
			src: `package foo

func f() {
	if true {
		for i := 0; i < 10; i++ {
			if true {
				x := 1
				_ = x
			}
		}
	}
}
`,
			maxDepth: 3,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name: "multiple functions independent",
			src: `package foo

func f() {
	if true {
		x := 1
		_ = x
	}
}

func g() {
	if true {
		x := 1
		_ = x
	}
}
`,
			maxDepth: 1,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name: "select stmt counts",
			src: `package foo

func f() {
	if true {
		select {
		case <-make(chan int):
			if true {
				x := 1
				_ = x
			}
		}
	}
}
`,
			maxDepth: 3,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name: "range stmt counts",
			src: `package foo

func f() {
	for range []int{1} {
		for range []int{1} {
			for range []int{1} {
				x := 1
				_ = x
			}
		}
	}
}
`,
			maxDepth: 2,
			wantN:    1,
			wantMsg: "nesting depth is 3 (limit 2)" +
				" — consider early returns" +
				" or extracting a helper",
		},
		{
			name: "type switch counts",
			src: `package foo

func f() {
	var x interface{} = 1
	if true {
		switch x.(type) {
		case int:
			if true {
				_ = x
			}
		}
	}
}
`,
			maxDepth: 2,
			wantN:    1,
			wantMsg: "nesting depth is 3 (limit 2)" +
				" — consider early returns" +
				" or extracting a helper",
		},
		{
			name: "comm clause body no extra depth",
			src: `package foo

func f() {
	ch := make(chan int, 1)
	ch <- 1
	select {
	case <-ch:
		if true {
			_ = 1
		}
	default:
		if true {
			_ = 2
		}
	}
}
`,
			maxDepth: 2,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name: "labeled statement",
			src: `package foo

func f() {
outer:
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			_ = i + j
			continue outer
		}
	}
}
`,
			maxDepth: 2,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name: "func literal nesting",
			src: `package foo

var f = func() {
	if true {
		if true {
			_ = 1
		}
	}
}
`,
			maxDepth: 1,
			wantN:    1,
			wantMsg: "nesting depth is 2 (limit 1)" +
				" — consider early returns" +
				" or extracting a helper",
		},
		{
			name: "nested func literal",
			src: `package foo

var f = func() {
	g := func() {
		if true {
			if true {
				_ = 1
			}
		}
	}
	_ = g
}
`,
			maxDepth: 1,
			wantN:    1,
			wantMsg: "nesting depth is 2 (limit 1)" +
				" — consider early returns" +
				" or extracting a helper",
		},
		{
			name:     "disabled when maxDepth zero",
			src:      `package foo; func f() { if true { if true { } } }`,
			maxDepth: 0,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name: "bare block statement",
			src: `package foo

func f() {
	{
		if true {
			if true {
				_ = 1
			}
		}
	}
}
`,
			maxDepth: 2,
			wantN:    0,
			wantMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, "test.go", f, tt.maxDepth)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}
