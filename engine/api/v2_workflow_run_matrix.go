package api

const (
	defaultMatrixParallelism    int64 = 20
	defaultMatrixMaxParallelism int64 = 40
)

// matrixParallelism returns how many permutations of a matrix job may run in parallel: the default applied when the
// job has no concurrency, and the maximum applied whatever its concurrency.
func (api *API) matrixParallelism() (int64, int64) {
	defaultParallelism := api.Config.WorkflowV2.MatrixDefaultParallelism
	if defaultParallelism <= 0 {
		defaultParallelism = defaultMatrixParallelism
	}
	maxParallelism := api.Config.WorkflowV2.MatrixMaxParallelism
	if maxParallelism <= 0 {
		maxParallelism = defaultMatrixMaxParallelism
	}
	if maxParallelism < defaultParallelism {
		maxParallelism = defaultParallelism
	}
	return defaultParallelism, maxParallelism
}
