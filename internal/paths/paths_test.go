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
