package main

import "testing"

func TestHelpAndVersion(t *testing.T) {
	if run([]string{"--help"}) != 0 {
		t.Fatal("help failed")
	}
	if run([]string{"--version"}) != 0 {
		t.Fatal("version failed")
	}
}
