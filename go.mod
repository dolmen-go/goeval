module github.com/dolmen-go/goeval

go 1.25.0

require (
	golang.org/x/mod v0.35.0
	golang.org/x/tools v0.44.0
)

require (
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/telemetry v0.0.0-20260409153401-be6f6cb8b1fa // indirect
)

tool (
	github.com/dolmen-go/goeval
	github.com/dolmen-go/goeval/internal/testexe/golden
	golang.org/x/tools/cmd/goimports
)
