module quad4/reticulum-go-mf

go 1.26.4

require (
	golang.org/x/term v0.43.0
	quad4/msgpack/v5 v5.8.0
	quad4/reticulum-go v0.9.7
)

require (
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	quad4/bzip2 v0.0.0 // indirect
	quad4/tagparser v0.0.0 // indirect
)

replace (
	quad4/bzip2 => github.com/Quad4-Software/bzip2 v0.0.0-20260704225916-ca8b2bb66059
	quad4/msgpack/v5 => github.com/Quad4-Software/msgpack/v5 v5.8.0
	quad4/pbt => github.com/Quad4-Software/pbt v0.0.0-20260614183135-abe0cfc4e604
	quad4/reticulum-go => /run/media/user1/projects/Reticulum/Reticulum-Go
	quad4/tagparser => github.com/Quad4-Software/tagparser v0.1.3-0.20260614183136-daa4d5f437ce
)
