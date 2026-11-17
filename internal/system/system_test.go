package system

import (
	"strings"
	"testing"
)

func TestParseOSRelease(t *testing.T) {
	in := `NAME="Fedora Linux"
VERSION="44 (Workstation Edition)"
ID=fedora
VERSION_ID=44
PRETTY_NAME="Fedora Linux 44 (Workstation Edition)"
VARIANT_ID=workstation
# comment
`
	o, err := ParseOSRelease(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := OSRelease{ID: "fedora", VersionID: "44", Variant: "workstation", Pretty: "Fedora Linux 44 (Workstation Edition)"}
	if o != want {
		t.Errorf("got %+v, want %+v", o, want)
	}
	if !o.IsFedora() {
		t.Error("IsFedora = false")
	}
}
