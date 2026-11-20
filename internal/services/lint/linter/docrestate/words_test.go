package docrestate

import (
	"reflect"
	"testing"
)

func TestSplitName_Words(t *testing.T) {
	assertSplit(t, "Run", "run")
	assertSplit(t, "NewClient", "new", "client")
	assertSplit(t, "max_size", "max", "size")
	assertSplit(t, "Base64Encode", "base", "64", "encode")
}

func TestSplitName_Acronyms(t *testing.T) {
	assertSplit(t, "HTTPServer", "http", "server")
	assertSplit(t, "GetUserID", "get", "user", "id")
	assertSplit(t, "UserIDs", "user", "ids")
	assertSplit(t, "URLs", "urls")
	assertSplit(t, "IDsFor", "ids", "for")
	assertSplit(t, "CPUUsage", "cpu", "usage")
}

func TestInflects_Matches(t *testing.T) {
	assertInflects(t, true,
		[2]string{"run", "run"}, [2]string{"runs", "run"},
		[2]string{"closes", "close"}, [2]string{"closing", "close"},
		[2]string{"parsed", "parse"}, [2]string{"loading", "load"},
		[2]string{"entries", "entry"}, [2]string{"strings", "string"},
		[2]string{"settings", "setting"}, [2]string{"needs", "need"},
		[2]string{"matches", "match"}, [2]string{"loaded", "load"},
		[2]string{"copied", "copy"}, [2]string{"used", "use"},
		[2]string{"boxes", "box"},
	)
}

func TestInflects_NonMatches(t *testing.T) {
	assertInflects(t, false,
		[2]string{"string", "str"}, [2]string{"str", "string"},
		[2]string{"card", "car"}, [2]string{"yes", "y"},
		[2]string{"user", "use"}, [2]string{"config", "configuration"},
	)
}

func TestHasSingleQuoted(t *testing.T) {
	assertSingleQuoted(t, true, "is 'a'", "is 'max size'", "is {'a'}", "it's 'a'")
	assertSingleQuoted(t, false, "it's Bob's", "is '' empty", "is 'open")
}

func assertSplit(t *testing.T, name string, want ...string) {
	t.Helper()
	if got := splitName(name); !reflect.DeepEqual(got, want) {
		t.Errorf("splitName(%q) = %v, want %v", name, got, want)
	}
}

func assertInflects(t *testing.T, want bool, pairs ...[2]string) {
	t.Helper()
	for _, p := range pairs {
		if got := inflects(p[0], p[1]); got != want {
			t.Errorf("inflects(%q, %q) = %v, want %v", p[0], p[1], got, want)
		}
	}
}

func assertSingleQuoted(t *testing.T, want bool, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if got := hasSingleQuoted(text); got != want {
			t.Errorf("hasSingleQuoted(%q) = %v, want %v", text, got, want)
		}
	}
}
