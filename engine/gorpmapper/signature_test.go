package gorpmapper

import (
	"bytes"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

type testService struct {
	sdk.Service
	SignedEntity
}

func (s testService) Canonical() CanonicalForms {
	return []CanonicalForm{
		"{{.ID}}{{.Name}}{{.Type}}{{.Region}}{{.IgnoreJobWithNoRegion}}",
	}
}

func Test_CanonicalFormWithPointer(t *testing.T) {
	m := New()
	m.Register(m.NewTableMapping(testService{}, "service", false, "id"))

	region := "test"

	cases := []struct {
		name string
		s    testService
		res  string
	}{
		{
			name: "Service with empty values",
			s: testService{
				Service: sdk.Service{
					CanonicalService: sdk.CanonicalService{
						ID:                    123,
						Name:                  "my-service",
						Type:                  sdk.TypeHatchery,
						Region:                nil,
						IgnoreJobWithNoRegion: nil,
					},
				},
			},
			res: "123my-servicehatchery<nil><nil>",
		},
		{
			name: "Service without empty values",
			s: testService{
				Service: sdk.Service{
					CanonicalService: sdk.CanonicalService{
						ID:                    123,
						Name:                  "my-service",
						Type:                  sdk.TypeHatchery,
						Region:                &region,
						IgnoreJobWithNoRegion: &sdk.True,
					},
				},
			},
			res: "123my-servicehatcherytesttrue",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _ := c.s.Canonical().Latest()
			tmpl, err := m.getCanonicalTemplate(f)
			require.NoError(t, err)
			require.NotNil(t, tmpl)

			var clearContent = new(bytes.Buffer)
			require.NoError(t, tmpl.Execute(clearContent, c.s))

			require.Equal(t, c.res, clearContent.String())
		})
	}
}
