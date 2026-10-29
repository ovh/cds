package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A microvm job is sized by the most specific flavor: the one the job asks
// for in runs-on, then the one of its worker model, then the hatchery default.
func TestGetFlavorMicroVM(t *testing.T) {
	model := func(jobFlavor, modelFlavor string) WorkerStarterWorkerModel {
		return WorkerStarterWorkerModel{
			ModelV2:     &V2WorkerModel{Type: WorkerModelTypeMicroVM},
			MicroVMSpec: V2WorkerModelMicroVMSpec{Flavor: modelFlavor},
			Flavor:      jobFlavor,
		}
	}

	require.Equal(t, "L", model("L", "S").GetFlavor(nil, "XS"), "the job flavor wins over the model flavor")
	require.Equal(t, "S", model("", "S").GetFlavor(nil, "XS"), "the model flavor wins over the hatchery default")
	require.Equal(t, "XS", model("", "").GetFlavor(nil, "XS"))
}
