package linelen

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

type checkCase struct {
	src     string
	wantN   int
	wantMsg string
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
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
}

func TestCheck_ShortLine(t *testing.T) {
	runCheckCase(t, checkCase{
		src:     "package foo\n\nfunc main() {}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_Exactly88Chars(t *testing.T) {
	runCheckCase(t, checkCase{
		src:     "package foo\n\n// " + strings.Repeat("a", 88-3) + "\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_Over88Chars(t *testing.T) {
	runCheckCase(t, checkCase{
		src:     "package foo\n\n// " + strings.Repeat("a", 89-3) + "\n",
		wantN:   1,
		wantMsg: "line is 89 columns wide, exceeds 88-column limit",
	})
}

func TestCheck_TabsExpandingPastLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"var x = 0 //\t\t\t\t\t\t\t\t\t\t\tcomment\n",
		wantN:   1,
		wantMsg: "line is 103 columns wide, exceeds 88-column limit",
	})
}

func TestCheck_URLLineExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"// See https://example.com/very/long/path/that/exceeds/the/limit/by/a/lot/aaaaaaaaaaaaaaaaaa\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_ImportPathExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\nimport (\n" +
			"\t\"github.com/mirurobotics/very/long/" +
			"package/path/that/would/exceed/" +
			"the/line/length/limit/easily\"\n" +
			")\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_StringLiteralExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\nvar x = " +
			"\"this is a very long string literal that " +
			"exceeds the eighty eight character " +
			"column limit for sure\"\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_BacktickStringLiteralExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\nvar x = " +
			"`this is a very long backtick string " +
			"literal that exceeds the eighty eight " +
			"character column limit`\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NonStringLongLineNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func reallyLongFunctionName(" +
			"parameterOne int, parameterTwo int, " +
			"parameterThree int) error {\n" +
			"\treturn nil\n}\n",
		wantN:   1,
		wantMsg: "line is 91 columns wide, exceeds 88-column limit",
	})
}

func TestCheck_HttpURLExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"// See http://example.com/very/long/" +
			"path/that/exceeds/the/limit/by/" +
			"a/lot/aaaaaaaaaaaaaaaaaa\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_StructTagWithJsonExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"type Foo struct {\n" +
			"\tName string " +
			"`json:\"name\" " +
			"gorm:\"column:name;not null;" +
			"uniqueIndex:unique_workspace_name\"`\n" +
			"}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_StructTagWithGormExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"type Foo struct {\n" +
			"\tWsp *Workspace " +
			"`gorm:\"foreignKey:WorkspaceID;" +
			"references:ID;" +
			"constraint:OnDelete:CASCADE\"`\n" +
			"}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_ParameterlessFuncWithReceiverAndReturnType(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func (c *ConfigInstanceList) " +
			"OrderByToSortDBItemAscFn() " +
			"map[string]utils.OrderByAscFn[cfgsdbmdls.ConfigInstance] {\n" +
			"\treturn nil\n}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_ParameterlessFuncWithReceiverNoReturn(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func (c *VeryLongReceiverStructName) " +
			"VeryLongMethodNameThatExceedsTheLimit() {\n" +
			"}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_ParameterlessFuncWithoutReceiver(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func VeryLongStandaloneFunctionName" +
			"ThatExceedsTheEightyEightColumnLimit() error {\n" +
			"\treturn nil\n}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_ParameterlessFuncMultipleReturns(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func (s *SomeVeryLongStruct) " +
			"MethodName() (SomeLongReturnType, error) {\n" +
			"\treturn SomeLongReturnType{}, nil\n}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_FuncWithFuncCallbackParamNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func (s *VeryLongServerStructName) " +
			"HandleSomeLongName(handler func() error, " +
			"name string) error {\n" +
			"\treturn nil\n}\n",
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_FuncWithSingleFuncParamNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func (s *VeryLongServerStructName) " +
			"HandleSomethingVeryLongXX(handler func() bool) " +
			"error {\n" +
			"\treturn nil\n}\n",
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_FuncWithParamsNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func reallyLongFunctionNameHereXX(" +
			"parameterOne int, parameterTwo int, " +
			"parameterThree int) error {\n" +
			"\treturn nil\n}\n",
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_NonFuncLongLineNotExemptByFuncRule(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"// This is a very long comment that has " +
			"nothing to do with func signatures and " +
			"exceeds the limit\n",
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_SingleLineFunctionBodyNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func (r *VeryLongReceiverStructName) " +
			"VeryLongMethodNameThatExceedsLimitX() " +
			"{ return nil }\n",
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_ParameterlessFuncReturningStructExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\n" +
			"func (r *VeryLongReceiverStructName) " +
			"VeryLongMethodNameExceedingLimit() " +
			"chan struct{} {\n" +
			"\treturn nil\n}\n",
		wantN:   0,
		wantMsg: "",
	})
}
