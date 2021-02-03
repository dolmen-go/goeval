
# goeval - Evaluate Go snippets instantly from the command line

## Demo

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

## Install

    $ go get -u github.com/dolmen-go/goeval

## How does it work?

`goeval` just wraps your code with the necessary text to build a `main` package and a `main` func with the given imports, pass it through the [`goimports` tool](https://godoc.org/golang.org/x/tools/cmd/goimports) (to automatically add missing imports), writes in a temporary file and calls `go run` with [`GO111MODULE=off`](https://golang.org/ref/mod#mod-commands).

`goimports` is enabled by default, but you can disable it to force explicit imports (for forward safety):

    $ goeval -goimports= -i fmt 'fmt.Println("Hello, world!")'
    Hello, world!

## Alternatives

* [gommand](https://github.com/sno6/gommand) Go one liner program. Similar to `python -c`.
* [gorram](https://github.com/natefinch/gorram) Like `go run` for any Go function.
* [goexec](https://github.com/shurcooL/goexec) A command line tool to execute Go functions.

## License

Copyright 2019 Olivier Mengué

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
