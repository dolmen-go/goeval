
# goeval - Evaluate Go snippets instantly from the command line

## Demo

```console
$ goeval 'fmt.Println("Hello, world!")'
Hello, world!
$ goeval 'fmt.Println(os.Args[1])' 'Hello, world!'
Hello, world!
$ goeval -i .=fmt -i os 'Println(os.Args[1])' 'Hello, world!'
Hello, world!
$ goeval -i math/rand 'fmt.Println(rand.Int())'
5577006791947779410

$ goeval -i fmt -i math/big -i os 'var x, y, z big.Int; x.SetString(os.Args[1], 10); y.SetString(os.Args[2], 10); fmt.Println(z.Mul(&x, &y).String())' 45673432245678899065433367889424354 136762347343433356789893322
6246405805150306996814033892780381988744339134177555648763988

$ goeval 'fmt.Printf("%x\n", sha256.Sum256([]byte(os.Args[1])))' abc
ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad

$ GO111MODULE=off go get github.com/klauspost/cpuid && goeval -i github.com/klauspost/cpuid/v2 'fmt.Println(cpuid.CPU.X64Level())'
3

$ goeval 'http.Handle("/",http.FileServer(http.Dir(".")));http.ListenAndServe(":8084",nil)'
```

### Go modules

Use `-i <module>@<version>` to import a Go module.

Use `-i <alias>=<module>@<version>` to import a Go module and import the package with the given alias.

```console
$ goeval -i .=github.com/bitfield/script@v0.21.4 'Exec("ls").Stdout()'
LICENSE
README.md
go.mod
go.sum
goeval
main.go

$ goeval -i github.com/klauspost/cpuid/v2@v2.2.3 -i github.com/klauspost/cpuid/v2 'fmt.Println(cpuid.CPU.X64Level())'
3
$ goeval -i cpuid=github.com/klauspost/cpuid/v2@v2.2.3 'fmt.Println(cpuid.CPU.X64Level())'
3
```

## Install

```console
$ go install github.com/dolmen-go/goeval@latest
```

## Uninstall

```console
$ go clean -i github.com/dolmen-go/goeval
```

## How does it work?

### GOPATH mode

`goeval` just wraps your code with the necessary text to build a `main` package and a `main` func with the given imports, pass it through the [`goimports` tool](https://godoc.org/golang.org/x/tools/cmd/goimports) (to automatically add missing imports), writes in a temporary file and calls `go run` with [`GO111MODULE=off`](https://golang.org/ref/mod#mod-commands).

`goimports` is enabled by default, but you can disable it to force explicit imports (for forward safety):

```console
$ goeval -goimports= -i fmt 'fmt.Println("Hello, world!")'
Hello, world!
```

### Go module mode

When at least one `module@version` is imported with `-i`, Go module mode is enabled. Two files are generated: `tmpxxxx.go` and `go.mod`. Then `go get .` is run to resolve and fetch dependencies, and then `go run`.

## Debugging

To debug a syntax error:

```console
$ goeval -E -goimports= ... | goimports
````

## Unsupported tricks

Here are some tricks that have worked in the past, that may still work in the last version, but are not guaranteed to work later.

### Use functions

The supported way:

```console
$ goeval 'var fact func(int)int;fact=func(n int)int{if n==1{return 1};return n*fact(n-1);};fmt.Println(fact(5))'
```

The hacky way:

```console
$ goeval 'fmt.Println(fact(5));};func fact(n int)int{if n==1{return 1};return n*fact(n-1)'
```

### Use generics

Needs:
- goeval compiled with Go 1.18+
- Go 1.18+ installed.

```console
$ goeval 'p(1);p("a");};func p[T any](x T){fmt.Println(x)'
1
a
$ goeval 'p(1);p(2.0);};func p[T int|float64](x T){x++;fmt.Println(x)'
2
3
$ goeval -i golang.org/x/exp/constraints 'p(1);p(2.0);};func p[T constraints.Signed|constraints.Float](x T){x++;fmt.Println(x)'
2
3
```

## Alternatives

* [gommand](https://github.com/sno6/gommand) Go one liner program. Similar to `python -c`.
* [gorram](https://github.com/natefinch/gorram) Like `go run` for any Go function.
* [goexec](https://github.com/shurcooL/goexec) A command line tool to execute Go functions.

## License

Copyright 2019-2023 Olivier Mengué

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
