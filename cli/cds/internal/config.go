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

	// ConfigFile is the path to the configfile
	ConfigFile string

	// NoWarnings set to True wont display warnings
	NoWarnings bool

	// InsecureSkipVerifyTLS to set sdk "CDS_SKIP_VERIFY" viper variable
	InsecureSkipVerifyTLS bool
)
