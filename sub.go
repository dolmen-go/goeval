package main

import _ "embed"

var (
	//go:embed sub/play/play.go
	playClient string
	//go:embed sub/share/share.go
	shareClient string
)
