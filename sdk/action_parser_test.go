package sdk

import (
	"context"
	"fmt"
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
		result       interface{}
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
			result: 1,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"commitID": 1,
				},
			},
		},
		{
			name:   "simple boolean variable",
			input:  "${{ git.closed }}",
			result: true,
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
				"job": map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, result)
			}
		})
	}
}

func TestParserOperations(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	tests := []struct {
		name         string
		context      map[string]interface{}
		input        string
		result       interface{}
		containError string
	}{
		{
			name:   "simple equality",
			input:  "${{ git.branch == 'master' }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
		{
			name:   "full variable equality",
			input:  "${{ git.branch == git.ref.branch }}",
			result: true,
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
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
		{
			name:   "full variable not equals",
			input:  "${{ git.branch != git.ref.branch }}",
			result: true,
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
			result: true,
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"num": 5,
				},
			},
		},
		{
			name:   "simple greater than",
			input:  "${{ job.num > job.num2 }}",
			result: true,
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
			result: true,
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
			result: true,
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
			result: true,
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"num":  1,
					"num2": 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, result)
			}
		})
	}
}

func TestParserBooleanExpression(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	tests := []struct {
		name         string
		context      map[string]interface{}
		input        string
		result       interface{}
		containError string
	}{
		{
			name:   "or expression true",
			input:  "${{ git.branch == 'master' || git.ref == 'testing' }}",
			result: true,
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
			result: false,
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
			result: true,
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
			result: false,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "prod",
				},
			},
		},
		{
			name:   "term expression or - true",
			input:  "${{ (git.branch == 'master' && git.ref == 'testing') || (git.branch == 'master' && git.ref == 'prod') }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "prod",
				},
			},
		},
		{
			name:   "term expression or - false",
			input:  "${{ (git.branch == 'dev' && git.id < 2) || (git.branch == 'dev' && git.ref == 'prod') }}",
			result: false,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "testing",
					"id":     3,
				},
			},
		},
		{
			name:   "term expression and - true",
			input:  "${{ (git.branch == 'master' || git.branch == 'testing') && (git.ref == 'testing' || git.id == 2) }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
					"ref":    "prod",
					"id":     2,
				},
			},
		},
		{
			name:   "term expression and - false",
			input:  "${{ (git.branch == 'master' || git.branch == 'testing') && (git.ref == 'testing' || git.id == 2) }}",
			result: false,
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
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, result)
			}
		})
	}
}

func TestInterpolate(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	input := `echo ${{git.author}}
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

func TestParserFuncContains(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	tests := []struct {
		name         string
		context      map[string]interface{}
		input        string
		result       interface{}
		containError string
	}{
		{
			name:   "simple contains function",
			input:  "${{ contains(git.branch, 'ast') }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
		{
			name:   "simple contains function with string filter",
			input:  "${{ contains(git.changes.*.message, 'foo') }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"changes": []map[string]interface{}{
						{
							"message": "not me",
						},
						{
							"message": "foo",
						},
						{
							"message": "not me",
						},
					},
				},
			},
		},
		{
			name:   "simple contains function with int filter",
			input:  "${{ contains(git.changes.*.message, '2') }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"changes": []map[string]interface{}{
						{
							"message": 0,
						},
						{
							"message": 2,
						},
						{
							"message": 1,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, result)
			}
		})
	}
}

func TestParserFuncStartsWith(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	tests := []struct {
		name         string
		context      map[string]interface{}
		input        string
		result       interface{}
		containError string
	}{
		{
			name:   "simple startsWith function",
			input:  "${{ startsWith(git.branch, 'Mas') }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
		{
			name:   "complex startsWith function",
			input:  "${{ startsWith(git.branch, git.prefix) }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "Master",
					"prefix": "mas",
				},
			},
		},
		{
			name:         "startsWith wrong input",
			input:        "${{ startsWith(git.branch, 2) }}",
			result:       "",
			containError: "searchValue argument must be a string",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "Master",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, result)
			}
		})
	}
}

func TestParserFuncEndsWith(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine)
	tests := []struct {
		name         string
		context      map[string]interface{}
		input        string
		result       interface{}
		containError string
	}{
		{
			name:   "simple endsWith function",
			input:  "${{ endsWith(git.branch, 'Ter') }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
		{
			name:   "complex endsWith function",
			input:  "${{ endsWith(git.branch, git.prefix) }}",
			result: true,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "mastEr",
					"prefix": "ter",
				},
			},
		},
		{
			name:         "endsWith wrong input",
			input:        "${{ endsWith(git.branch, 2) }}",
			result:       "",
			containError: "searchValue argument must be a string",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "Master",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, result)
			}
		})
	}
}

func TestParserFuncFormat(t *testing.T) {
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
			name:    "simple string format",
			input:   "${{ format('Hello {0} {1}', 'foo', 'bar') }}",
			result:  "Hello foo bar",
			context: map[string]interface{}{},
		},
		{
			name:   "complex string format with variable",
			input:  "${{ format(job.message, job.replace1, job.replace2) }}",
			result: "Hello foo bar",
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"message":  "Hello {0} {1}",
					"replace1": "foo",
					"replace2": "bar",
				},
			},
		},
		{
			name:   "complex string and int format with variable",
			input:  "${{ format(job.message, job.replace1, job.replace2) }}",
			result: "Hello foo 2",
			context: map[string]interface{}{
				"job": map[string]interface{}{
					"message":  "Hello {0} {1}",
					"replace1": "foo",
					"replace2": 2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, result)
			}
		})
	}
}

func TestParserFuncJoin(t *testing.T) {
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
			name:   "object filter join ",
			input:  "${{ join(git.changes.*.id, '-') }}",
			result: "1-2-3",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"changes": []map[string]interface{}{
						{
							"id": 1,
						},
						{
							"id": 2,
						},
						{
							"id": 3,
						},
					},
				},
			},
		},
		{
			name:   "join with no separator",
			input:  "${{ join(git.changes.*.id) }}",
			result: "1,2,3",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"changes": []map[string]interface{}{
						{
							"id": 1,
						},
						{
							"id": 2,
						},
						{
							"id": 3,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, result)
			}
		})
	}
}

func TestParserFuncToJSON(t *testing.T) {
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
			name:  "toJSON",
			input: "${{ toJSON(git) }}",
			result: `{
  "branch": "master"
}`,
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"branch": "master",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, result)
			}
		})
	}
}

func TestParserReturningObject(t *testing.T) {
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
			name:   "parse returning object",
			input:  "${{ git.repo }}",
			result: "map[string]interface {}",
			context: map[string]interface{}{
				"git": map[string]interface{}{
					"repo": map[string]interface{}{
						"id":  "123",
						"url": "http://lolcat?host",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := NewActionParser(tt.context, DefaultFuncs)
			result, err := ap.parse(context.TODO(), tt.input)
			if tt.containError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.containError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.result, fmt.Sprintf("%T", result))
			}
		})
	}
}
