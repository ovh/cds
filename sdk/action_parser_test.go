package sdk

import (
	"context"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParserValidate(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)

	input := `echo ${{ git.author }}
if [[ "${{ ${{git.branch }}" == 'master' ]]; then
  echo 'Master branch'
fi;
echo "Job in progress: ${{ job.status == 'in progress' }}"
echo "First job: ${{ job.buildNumber < 2 }}"
echo "Commit message contains foobar: ${{ contains(git.message, 'foobar' }}"
`

	ap := NewActionParser(nil, nil)
	err := ap.Validate(context.TODO(), input)
	t.Logf("%v", err)
	require.Error(t, err)
}

func TestParserVariables(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)

	tests := []struct {
		name         string
		context      map[string]interface{}
		input        string
		result       string
		containError string
	}{
		{
			name:   "simple string variable",
			input:  "${{ git.branch }}",
			result: "master",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
		{
			name:   "deeper string variable",
			input:  "${{ job.step.line.column }}",
			result: "foo",
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"step": map[string]interface{}{
						"line": map[string]interface{}{
							"column": "foo",
						},
					},
				},
			},
		},
		{
			name:   "simple number variable",
			input:  "${{ git.commitID }}",
			result: "1",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"commitID": 1,
				},
			},
		},
		{
			name:   "simple boolean variable",
			input:  "${{ git.closed }}",
			result: "true",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"closed": true,
				},
			},
		},
		{
			name:   "simple array variable",
			input:  "${{ git.changes[0].hash }}",
			result: "123456",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"changes": []map[string]interface{}{
						{
							"hash": "123456",
						},
					},
				},
			},
		},
		{
			name:   "simple array variable",
			input:  "${{ git.changes[0].hash }}",
			result: "123456",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"changes": []map[string]interface{}{
						{
							"hash": "123456",
						},
					},
				},
			},
		},
		{
			name:         "unknown context",
			input:        "${{ job.id }}",
			result:       "",
			containError: "unknown context job",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
		{
			name:         "unknown variable in context must return empty",
			input:        "${{ job.id }}",
			result:       "",
			containError: "",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
	}

	for _, tt := range tests {
		ap := NewActionParser(tt.context, DefaultFuncs)
		result, err := ap.parse(context.TODO(), tt.input)
		if tt.containError != "" {
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.containError)
		} else {
			require.Equal(t, tt.result, result)
		}
	}
}

func TestParserOperations(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	tests := []struct {
		name         string
		context      map[string]interface{}
		input        string
		result       string
		containError string
	}{
		{
			name:   "simple equality",
			input:  "${{ git.branch == 'master' }}",
			result: "true",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
		{
			name:   "full variable equality",
			input:  "${{ git.branch == git.ref.branch }}",
			result: "true",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref": map[string]interface{}{
						"branch": "master",
					},
				},
			},
		},
		{
			name:   "simple variable not equals",
			input:  "${{ git.branch != 'dev/myfeature' }}",
			result: "true",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
		{
			name:   "full variable not equals",
			input:  "${{ git.branch != git.ref.branch }}",
			result: "true",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref": map[string]interface{}{
						"branch": "dev/myfeature",
					},
				},
			},
		},
		{
			name:   "simple greater than",
			input:  "${{ job.num > 1 }}",
			result: "true",
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"num": 5,
				},
			},
		},
		{
			name:   "simple greater than",
			input:  "${{ job.num > job.num2 }}",
			result: "true",
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"num":  5,
					"num2": 1,
				},
			},
		},
		{
			name:   "simple greater or equal than",
			input:  "${{ job.num >= job.num2 }}",
			result: "true",
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"num":  5,
					"num2": 5,
				},
			},
		},
		{
			name:   "simple less than",
			input:  "${{ job.num < job.num2 }}",
			result: "true",
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"num":  1,
					"num2": 5,
				},
			},
		},
		{
			name:   "simple less or equal than",
			input:  "${{ job.num <= job.num2 }}",
			result: "true",
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"num":  1,
					"num2": 1,
				},
			},
		},
	}

	for _, tt := range tests {
		ap := NewActionParser(tt.context, DefaultFuncs)
		result, err := ap.parse(context.TODO(), tt.input)
		if tt.containError != "" {
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.containError)
		} else {
			require.Equal(t, tt.result, result)
		}
	}
}

func TestParserBooleanExpression(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	tests := []struct {
		name         string
		context      map[string]interface{}
		input        string
		result       string
		containError string
	}{
		{
			name:   "or expression true",
			input:  "${{ git.branch == 'master' || git.ref == 'testing' }}",
			result: "true",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "prod",
				},
			},
		},
		{
			name:   "or expression false",
			input:  "${{ git.branch == 'master' || git.ref == 'testing' }}",
			result: "false",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "dev",
					"ref":    "prod",
				},
			},
		},

		{
			name:   "and expression true",
			input:  "${{ git.branch == 'master' && git.ref == 'testing' }}",
			result: "true",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "testing",
				},
			},
		},
		{
			name:   "and expression false",
			input:  "${{ git.branch == 'master' && git.ref == 'testing' }}",
			result: "false",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "prod",
				},
			},
		},
		{
			name:   "term expression or - ok",
			input:  "${{ (git.branch == 'master' && git.ref == 'testing') || (git.branch == 'master' && git.ref == 'prod') }}",
			result: "true",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "prod",
				},
			},
		},
		{
			name:   "term expression or - ko",
			input:  "${{ (git.branch == 'dev' && git.id < 2) || (git.branch == 'dev' && git.ref == 'prod') }}",
			result: "false",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "testing",
					"id":     3,
				},
			},
		},
		{
			name:   "term expression and - ok",
			input:  "${{ (git.branch == 'master' || git.branch == 'testing') && (git.ref == 'testing' || git.id == 2) }}",
			result: "true",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "prod",
					"id":     2,
				},
			},
		},
		{
			name:   "term expression and - ko",
			input:  "${{ (git.branch == 'master' || git.branch == 'testing') && (git.ref == 'testing' || git.id == 2) }}",
			result: "false",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "prod",
					"id":     3,
				},
			},
		},
	}

	for _, tt := range tests {
		ap := NewActionParser(tt.context, DefaultFuncs)
		result, err := ap.parse(context.TODO(), tt.input)
		if tt.containError != "" {
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.containError)
		} else {
			require.Equal(t, tt.result, result)
		}
	}
}

func TestInterpolate(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	input := `echo ${{ git.author }}
if [[ "${{ git.branch }}" == 'master' ]]; then
  echo 'Master branch'
fi;
echo "Job in progress: ${{ job.status == 'in progress' }}"
echo "First job: ${{ job.buildNumber < 2 }}"
echo "Commit message contains foobar: ${{ contains(git.message, 'foobar') }}"
`

	result := `echo Steven
if [[ "master" == 'master' ]]; then
  echo 'Master branch'
fi;
echo "Job in progress: true"
echo "First job: true"
echo "Commit message contains foobar: true"
`

	actionCtx := map[string]interface{}{
		"git": map[string]interface{}{
			"branch":  "master",
			"author":  "Steven",
			"message": "Message foobar, feature ascode",
		},
		"job": map[string]interface{}{
			"status":      "in progress",
			"buildNumber": 1,
		},
	}

	app := NewActionParser(actionCtx, DefaultFuncs)
	interpolatedInput, err := app.Interpolate(context.TODO(), input)
	require.NoError(t, err)
	require.Equal(t, result, interpolatedInput)

}

func TestInterpolateError(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	input := `if [[ "${{ ${{git.branch }}" == 'master' ]]; then`

	app := NewActionParser(nil, DefaultFuncs)
	_, err := app.Interpolate(context.TODO(), input)
	t.Logf("%v", err)
	require.Error(t, err)
}
