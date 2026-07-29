package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func Test_useBareAnalysisCache(t *testing.T) {
	analysisOp := func() sdk.Operation {
		op := sdk.Operation{}
		op.Setup.Checkout.Branch = "master"
		op.Setup.Checkout.CheckSignature = true
		op.Setup.Checkout.ProcessSemver = true
		op.Setup.Checkout.GetChangeSet = true
		return op
	}

	tests := []struct {
		name   string
		flag   bool
		mutate func(op *sdk.Operation)
		want   bool
	}{
		{name: "analysis operation on branch", flag: true, want: true},
		{name: "flag disabled keeps current path", flag: false, want: false},
		{name: "analysis operation on tag", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Checkout.Branch = ""
			op.Setup.Checkout.Tag = "v1.0.0"
		}, want: true},
		{name: "single analysis flag is enough", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Checkout.CheckSignature = false
			op.Setup.Checkout.GetChangeSet = false
		}, want: true},
		{name: "loadfiles operation keeps current path", flag: true, mutate: func(op *sdk.Operation) {
			op.LoadFiles.Pattern = ".cds/*.yml"
		}, want: false},
		{name: "push operation keeps current path", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Push.FromBranch = "cds/update"
		}, want: false},
		{name: "no branch nor tag keeps current path", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Checkout.Branch = ""
		}, want: false},
		{name: "no analysis flag keeps current path", flag: true, mutate: func(op *sdk.Operation) {
			op.Setup.Checkout.CheckSignature = false
			op.Setup.Checkout.ProcessSemver = false
			op.Setup.Checkout.GetChangeSet = false
			op.Setup.Checkout.GetMessage = true
		}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{}
			s.Cfg.BareAnalysisCache = tt.flag
			op := analysisOp()
			if tt.mutate != nil {
				tt.mutate(&op)
			}
			assert.Equal(t, tt.want, s.useBareAnalysisCache(op))
		})
	}
}
