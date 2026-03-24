#!/usr/bin/env bash
#
# Run an integration testsuite and collect code coverage.
#
# Reference documentation: https://go.dev/blog/integration-test-coverage


set -euo pipefail

output="${1:-.coverage.out}"
go=go
goeval=./goeval.cover
goeval_offline=./goeval-offline.cover

if [[ ! -d .coverage.1 || ! -d .coverage.2 ]]; then
        mkdir -p .coverage.1 .coverage.2
fi
rm -f .coverage.1/cov* .coverage.2/cov* .coverage/cov* "$output"


unset GO111MODULE

echo Building...

$go build -cover -coverpkg=./... -buildvcs=true -o=$goeval .
$go build -cover -coverpkg=./... -buildvcs=true -tags=goeval.offline -o=$goeval_offline .

echo '== Testsuite =='

GOCOVERDIR=$(pwd)/.coverage.1

# Ensure that goimports (declared as tool in go.mod) is built
$go tool goimports -h >/dev/null || :
# Show goimports version
$go version -m "$($go tool -n goimports)"

export GOCOVERDIR

echo Running tests...

# --------------------------------------------------------------

$goeval || :
$goeval -h || :

$goeval 'fmt.Println("Hello, world")'

# stdin
$goeval - <<EOF
fmt.Println("Hello, world")
EOF

# stdin with shebang
$goeval - <<EOF
#!/usr/bin/env goeval -
fmt.Println("Hello, world")
EOF


$goeval -E 'fmt.Println("Hello, world")'

$goeval -x 'fmt.Println("Hello, world")'


# -i with comma
# -i with . as alias
# -i with _ as alias
$goeval -i=.=fmt,time,_=os 'Println(time.Now())'

# -o
{
        tmp_exe=$(mktemp -t .goeval-cover.XXXXXXXXX)
        $goeval -i=.=fmt,time -o "$tmp_exe" 'Println(time.Now())'
        "$tmp_exe"
        rm "$tmp_exe"
}

# With goimports builtin
$goeval -goimports=goimports -E 'fmt.Println("Hello, world")'

# Without goimports builtin
$goeval -goimports= -E 'fmt.Println("Hello, world")'

# With goimports as external command
# We are using the one declared as a tool in go.mod
$goeval -goimports="$($go tool -n goimports)" -E 'fmt.Println("Hello, world")'


# GOPATH mode with external package
# With go1.26, "go get" doesn't work anymore in GOPATH mode
#GO111MODULE=off $go get github.com/klauspost/cpuid
if [[ ! -d "$(go env GOPATH)"/src/github.com/klauspost/cpuid/v2 ]]; then
        mkdir -p "$(go env GOPATH)"/src/github.com/klauspost/cpuid
        git -C "$(go env GOPATH)"/src/github.com/klauspost/cpuid clone https://github.com/klauspost/cpuid.git v2
fi
GO111MODULE=off $goeval -i github.com/klauspost/cpuid/v2 'fmt.Println(cpuid.CPU.X64Level())'

# Go module mode with external package
$goeval -i cpuid=github.com/klauspost/cpuid/v2@v2.3.0 'fmt.Println(cpuid.CPU.X64Level())'

# -E with Go module mode with external package
$goeval -E -i cpuid=github.com/klauspost/cpuid/v2@v2.3.0 'fmt.Println(cpuid.CPU.X64Level())'

$goeval -Eplay 'fmt.Println("Hello, world")'

# -Eplay with Go module mode with external package
$goeval -Eplay -i cpuid=github.com/klauspost/cpuid/v2@v2.3.0 'fmt.Println(cpuid.CPU.X64Level())'


$goeval -play 'fmt.Println("Hello, world")'

# -play with os.Args
$goeval -play 'fmt.Println(os.Args[1])' 'Hello, world'

$goeval_offline -h || :

$goeval_offline 'fmt.Println("Hello, world")'

$goeval_offline -play 'fmt.Println("Hello, world")' || :

# We don't test -share here to not pollute play.golang.org
# TODO(dolmen) Mock play.golang.org/share

# -------------------------------------------------------------

# Tests in sub/play_test.go and sub/share_test.go manage production of coverage data
# based on the presence of the GOCOVERDIR value.
# So we must run "normal" tests (not -cover), but they'll receive our global GOCOVERDIR.

$go test -v ./sub/play ./sub/share ./internal/testexe/...

unset GOCOVERDIR


echo '== go test ./... =='
# Coverage of internal/testexe via testsuite in internal/testexe/echo
# => .coverage.2
$go test -cover -coverpkg=./... ./... -args -test.gocoverdir="$(pwd)"/.coverage.2


# -------------------------------------------------------------

echo '== Aggreated coverage =='

[ -d .coverage ] || mkdir .coverage
go tool covdata merge -i=.coverage.1,.coverage.2 -o=.coverage

# Show aggregated percent
go tool covdata percent -i=.coverage

go tool covdata textfmt -i=.coverage -o="$output"

if [[ -t 0 ]] && command -v open >/dev/null; then
        $go tool cover -html="$output"
fi
