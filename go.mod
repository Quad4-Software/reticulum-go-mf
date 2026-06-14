module quad4/reticulum-go-mf

go 1.26.2

require (
	golang.org/x/term v0.42.0
	quad4/msgpack/v5 v5.0.0
	quad4/reticulum-go v0.0.0
)

require (
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	quad4/tagparser v0.0.0 // indirect
)

replace (
	quad4/bzip2 => ../bzip2
	quad4/msgpack/v5 => ../msgpack
	quad4/pbt => ../pbt
	quad4/reticulum-go => ../../Reticulum/Reticulum-Go
	quad4/tagparser => ../tagparser
)
