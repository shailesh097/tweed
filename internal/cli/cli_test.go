package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestMultiFlag(t *testing.T) {
	var m multiFlag
	for _, v := range []string{"base", "dev, apps", ",,"} {
		_ = m.Set(v)
	}
	if want := (multiFlag{"base", "dev", "apps"}); !reflect.DeepEqual(m, want) {
		t.Errorf("got %v, want %v", m, want)
	}
}

func TestListBuiltinConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // make sure no user config is picked up
	var out bytes.Buffer
	if err := runList(nil, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"built-in default", "workstation", "server"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("list output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if code := Run([]string{"frobnicate"}, "test"); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}
