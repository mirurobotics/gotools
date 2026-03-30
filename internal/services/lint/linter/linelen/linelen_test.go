package linelen

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	// Helper: 88 'a' characters.
	a88 := make([]byte, 88)
	for i := range a88 {
		a88[i] = 'a'
	}
	a89 := make([]byte, 89)
	for i := range a89 {
		a89[i] = 'a'
	}

	tests := []struct {
		name    string
		src     string
		wantN   int
		wantMsg string
	}{
		{
			name:    "short line",
			src:     "package foo\n\nfunc main() {}\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name:    "exactly 88 chars",
			src:     "package foo\n\n// " + string(a88[:88-3]) + "\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name:    "over 88 chars",
			src:     "package foo\n\n// " + string(a89[:89-3]) + "\n",
			wantN:   1,
			wantMsg: "line is 89 columns wide, exceeds 88-column limit",
		},
		{
			name: "tabs expanding past limit",
			src: "package foo\n\n" +
				"var x = 0 //\t\t\t\t\t\t\t\t\t\t\tcomment\n",
			wantN:   1,
			wantMsg: "line is 103 columns wide, exceeds 88-column limit",
		},
		{
			name: "URL line exempt",
			src: "package foo\n\n" +
				"// See https://example.com/very/long/path/that/exceeds/the/limit/by/a/lot/aaaaaaaaaaaaaaaaaa\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "import path exempt",
			src: "package foo\n\nimport (\n" +
				"\t\"github.com/mirurobotics/very/long/" +
				"package/path/that/would/exceed/" +
				"the/line/length/limit/easily\"\n" +
				")\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "string literal exempt",
			src: "package foo\n\nvar x = " +
				"\"this is a very long string literal that " +
				"exceeds the eighty eight character " +
				"column limit for sure\"\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "backtick string literal exempt",
			src: "package foo\n\nvar x = " +
				"`this is a very long backtick string " +
				"literal that exceeds the eighty eight " +
				"character column limit`\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "non-string long line not exempt",
			src: "package foo\n\n" +
				"func reallyLongFunctionName(" +
				"parameterOne int, parameterTwo int, " +
				"parameterThree int) error {\n" +
				"\treturn nil\n}\n",
			wantN:   1,
			wantMsg: "line is 91 columns wide, exceeds 88-column limit",
		},
		{
			name: "http URL exempt",
			src: "package foo\n\n" +
				"// See http://example.com/very/long/" +
				"path/that/exceeds/the/limit/by/" +
				"a/lot/aaaaaaaaaaaaaaaaaa\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "struct tag with json exempt",
			src: "package foo\n\n" +
				"type Foo struct {\n" +
				"\tName string " +
				"`json:\"name\" " +
				"gorm:\"column:name;not null;" +
				"uniqueIndex:unique_workspace_name\"`\n" +
				"}\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "struct tag with gorm exempt",
			src: "package foo\n\n" +
				"type Foo struct {\n" +
				"\tWsp *Workspace " +
				"`gorm:\"foreignKey:WorkspaceID;" +
				"references:ID;" +
				"constraint:OnDelete:CASCADE\"`\n" +
				"}\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "parameterless func with receiver and return type",
			src: "package foo\n\n" +
				"func (c *ConfigInstanceList) " +
				"OrderByToSortDBItemAscFn() " +
				"map[string]utils.OrderByAscFn[cfgsdbmdls.ConfigInstance] {\n" +
				"\treturn nil\n}\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "parameterless func with receiver no return",
			src: "package foo\n\n" +
				"func (c *VeryLongReceiverStructName) " +
				"VeryLongMethodNameThatExceedsTheLimit() {\n" +
				"}\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "parameterless func without receiver",
			src: "package foo\n\n" +
				"func VeryLongStandaloneFunctionName" +
				"ThatExceedsTheEightyEightColumnLimit() error {\n" +
				"\treturn nil\n}\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "parameterless func multiple returns",
			src: "package foo\n\n" +
				"func (s *SomeVeryLongStruct) " +
				"MethodName() (SomeLongReturnType, error) {\n" +
				"\treturn SomeLongReturnType{}, nil\n}\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "func with func() callback param not exempt",
			src: "package foo\n\n" +
				"func (s *VeryLongServerStructName) " +
				"HandleSomeLongName(handler func() error, " +
				"name string) error {\n" +
				"\treturn nil\n}\n",
			wantN:   1,
			wantMsg: "",
		},
		{
			name: "func with single func() param not exempt",
			src: "package foo\n\n" +
				"func (s *VeryLongServerStructName) " +
				"HandleSomethingVeryLongXX(handler func() bool) " +
				"error {\n" +
				"\treturn nil\n}\n",
			wantN:   1,
			wantMsg: "",
		},
		{
			name: "func with params not exempt",
			src: "package foo\n\n" +
				"func reallyLongFunctionNameHereXX(" +
				"parameterOne int, parameterTwo int, " +
				"parameterThree int) error {\n" +
				"\treturn nil\n}\n",
			wantN:   1,
			wantMsg: "",
		},
		{
			name: "non-func long line not exempt by func rule",
			src: "package foo\n\n" +
				"// This is a very long comment that has " +
				"nothing to do with func signatures and " +
				"exceeds the limit\n",
			wantN:   1,
			wantMsg: "",
		},
		{
			name: "single-line function body not exempt",
			src: "package foo\n\n" +
				"func (r *VeryLongReceiverStructName) " +
				"VeryLongMethodNameThatExceedsLimitX() " +
				"{ return nil }\n",
			wantN:   1,
			wantMsg: "",
		},
		{
			name: "parameterless func returning struct{} exempt",
			src: "package foo\n\n" +
				"func (r *VeryLongReceiverStructName) " +
				"VeryLongMethodNameExceedingLimit() " +
				"chan struct{} {\n" +
				"\treturn nil\n}\n",
			wantN:   0,
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, "test.go", f, []byte(tt.src), 88, 8)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}
