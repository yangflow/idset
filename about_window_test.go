package main

import "testing"

// mygo build makes every platform's icon from resources/icon.png, which must be a 1024 square.
func TestProductIcon(t *testing.T) {
	if w, h := productIcon().Size(); w != 1024 || h != 1024 {
		t.Errorf("resources/icon.png is %d x %d", w, h)
	}
}
