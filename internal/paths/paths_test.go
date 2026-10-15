package paths

import "testing"

func TestTildeAndExpand(t *testing.T) {
	unix := Env{Home: "/home/ada"}
	win := Env{Home: `C:\Users\Ada`, Windows: true}
	for _, c := range []struct {
		env         Env
		abs, stored string
	}{
		{unix, "/home/ada", "~"},
		{unix, "/home/ada/work/", "~/work/"},
		{unix, "/home/adam/x", "/home/adam/x"},
		{unix, "/opt/repos/", "/opt/repos/"},
		{win, `C:\Users\Ada\work`, "~/work"},
		{win, `c:\users\ada\Work\`, "~/Work/"},
		{win, `D:\src`, "D:/src"},
	} {
		if got := c.env.Tilde(c.abs); got != c.stored {
			t.Errorf("Tilde(%q) = %q, want %q", c.abs, got, c.stored)
		}
	}
	linked := Env{Home: "/tmp/ada", RealHome: "/private/tmp/ada"}
	if got := linked.Tilde("/private/tmp/ada/work"); got != "~/work" {
		t.Errorf("resolved home: %q", got)
	}
	if got := linked.Tilde("/tmp/ada/work"); got != "~/work" {
		t.Errorf("configured home: %q", got)
	}
	if got := unix.Expand("~/work/"); got != "/home/ada/work/" {
		t.Errorf("Expand = %q", got)
	}
	if got := win.Expand("~/work"); got != "C:/Users/Ada/work" {
		t.Errorf("Windows Expand = %q", got)
	}
	if got := unix.Expand("~bob/x"); got != "~bob/x" {
		t.Errorf("another user's home must not expand: %q", got)
	}
}

func TestSplitKeepsTheStartAndTheLastSegment(t *testing.T) {
	for _, c := range []struct{ path, lead, mid, last string }{
		{"~/Documents/archive/2023/legacy-services/payments-gateway/", "~/", "Documents/archive/2023/legacy-services/", "payments-gateway/"},
		{"~/work/", "~/", "", "work/"},
		{"~/work/api", "~/", "work/", "api"},
		{"~", "~", "", ""},
		{"~/", "~/", "", ""},
		{"/home/ada/work/api/", "/", "home/ada/work/", "api/"},
		{"/", "/", "", ""},
		{"/srv", "/", "", "srv"},
		{`C:\Users\Ada\work\api\`, "C:/", "Users/Ada/work/", "api/"},
		{"c:/work/", "c:/", "", "work/"},
		{"C:", "C:", "", ""},
		{"//server/share/team/api/", "//", "server/share/team/", "api/"},
		{"work/api/", "", "work/", "api/"},
		{"api", "", "", "api"},
		{"", "", "", ""},
		{"~tilde/x", "", "~tilde/", "x"},
	} {
		lead, mid, last := Split(c.path)
		if lead != c.lead || mid != c.mid || last != c.last {
			t.Errorf("%q: got %q %q %q, want %q %q %q", c.path, lead, mid, last, c.lead, c.mid, c.last)
		}
		if whole := lead + mid + last; whole != Slash(c.path) {
			t.Errorf("%q: the parts add up to %q", c.path, whole)
		}
	}
}
