module github.com/antonybholmes/go-mutations

go 1.25

replace github.com/antonybholmes/go-dna => ../go-dna

replace github.com/antonybholmes/go-basemath => ../go-basemath

replace github.com/antonybholmes/go-sys => ../go-sys

require (
	github.com/antonybholmes/go-dna v0.0.0-20250911221452-259e76605944
	github.com/rs/zerolog v1.34.0
)

require (
	github.com/antonybholmes/go-basemath v0.0.0-20250911171638-68021ab78342 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/exp v0.0.0-20250911091902-df9299821621 // indirect
	golang.org/x/sys v0.36.0 // indirect
)
