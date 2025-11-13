module github.com/antonybholmes/go-mutations

go 1.25

replace github.com/antonybholmes/go-dna => ../go-dna

replace github.com/antonybholmes/go-basemath => ../go-basemath

replace github.com/antonybholmes/go-sys => ../go-sys

require (
	github.com/antonybholmes/go-dna v0.0.0-20251112171523-6049b30cdbe8
	github.com/rs/zerolog v1.34.0
)

require (
	github.com/antonybholmes/go-basemath v0.0.0-20251107200700-4d494adc3d42 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/sys v0.38.0 // indirect
)
