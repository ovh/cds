package environment

import (
	"github.com/go-gorp/gorp"
)

// CreateBuiltinEnvironments creates default environment if needed
func CreateBuiltinEnvironments(db gorp.SqlExecutor) error {
	return CheckDefaultEnv(db)
}
