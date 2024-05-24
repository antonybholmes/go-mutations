module github.com/antonybholmes/go-mutations

go 1.22.2

replace github.com/antonybholmes/go-dna => ../go-dna

replace github.com/antonybholmes/go-sys => ../go-sys

require (
	github.com/antonybholmes/go-dna v0.0.0-20240503021126-08c3c39059f5
	github.com/antonybholmes/go-sys v0.0.0-20240501232923-152b6e4cc204
	github.com/rs/zerolog v1.33.0
)

require (
	github.com/antonybholmes/go-math v0.0.0-20240215163921-12bb7e52185c // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.20.0 // indirect
)
