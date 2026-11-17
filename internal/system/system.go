// Package system inspects the machine tweed is running on.
package system

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// OSRelease holds the fields of /etc/os-release that tweed cares about.
type OSRelease struct {
	ID        string // e.g. "fedora"
	VersionID string // e.g. "44"
	Variant   string // e.g. "workstation", "server"
	Pretty    string // e.g. "Fedora Linux 44 (Workstation Edition)"
}

// IsFedora reports whether the system is Fedora.
func (o OSRelease) IsFedora() bool { return o.ID == "fedora" }

// ReadOSRelease reads /etc/os-release.
func ReadOSRelease() (OSRelease, error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return OSRelease{}, err
	}
	defer f.Close()
	return ParseOSRelease(f)
}

// ParseOSRelease parses os-release(5) formatted data.
func ParseOSRelease(r io.Reader) (OSRelease, error) {
	var o OSRelease
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if !ok || strings.HasPrefix(k, "#") {
			continue
		}
		v = strings.Trim(v, `"'`)
		switch k {
		case "ID":
			o.ID = v
		case "VERSION_ID":
			o.VersionID = v
		case "VARIANT_ID":
			o.Variant = v
		case "PRETTY_NAME":
			o.Pretty = v
		}
	}
	return o, sc.Err()
}
