module github.com/dolmen-go/goeval

go 1.24.0

require (
	golang.org/x/mod v0.33.0
	golang.org/x/tools v0.42.0
)

require (
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/telemetry v0.0.0-20260209163413-e7419c687ee4 // indirect
)

tool (
	github.com/dolmen-go/goeval
	github.com/dolmen-go/goeval/internal/testexe/golden
	golang.org/x/tools/cmd/goimports
)
