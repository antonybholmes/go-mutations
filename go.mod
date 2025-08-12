module github.com/antonybholmes/go-mutations

go 1.24

toolchain go1.24.0

replace github.com/antonybholmes/go-dna => ../go-dna

replace github.com/antonybholmes/go-basemath => ../go-basemath

replace github.com/antonybholmes/go-sys => ../go-sys

require (
	github.com/antonybholmes/go-dna v0.0.0-20250731210107-8a77962e05e0
	github.com/rs/zerolog v1.34.0
)

require (
	github.com/antonybholmes/go-basemath v0.0.0-20250718220222-02e267b47e76 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/exp v0.0.0-20250811191247-51f88131bc50 // indirect
	golang.org/x/sys v0.35.0 // indirect
)
