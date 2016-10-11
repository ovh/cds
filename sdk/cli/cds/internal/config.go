package internal

var (
	// Verbose conditions the quantity of output of api requests
	Verbose bool

	// Architecture (linux-amd64, etc...), just for debug information
	Architecture string

	// DateCreation of binary
	DateCreation string

	// Sha1 of binary
	Sha1 string

	// PackagingInformations contains link to CI / CD system use for building
	PackagingInformations string
)
