module github.com/antonybholmes/go-mutations

go 1.24

replace github.com/antonybholmes/go-dna => ../go-dna

replace github.com/antonybholmes/go-basemath => ../go-basemath

replace github.com/antonybholmes/go-sys => ../go-sys

require (
	github.com/antonybholmes/go-dna v0.0.0-20250210230739-b3791e252d47
	github.com/rs/zerolog v1.33.0
)

require (
	github.com/antonybholmes/go-basemath v0.0.0-20250210230743-9da55b7ac3fd // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/exp v0.0.0-20250210185358-939b2ce775ac // indirect
	golang.org/x/sys v0.30.0 // indirect
)
