package swagger

type Schema struct {
	Type  string            `json:"type,omitempty"`
	Items map[string]string `json:"items,omitempty"`
	Ref   string            `json:"$ref,omitempty"`
}

type Response struct {
	Description string  `json:"description"`
	Schema      *Schema `json:"schema,omitempty"`
}

type Operation struct {
	HttpMethod       string              `json:"-"`
	Nickname         string              `json:"-"`
	Items            map[string]string   `json:"items,omitempty"`
	Type             string              `json:"-"`
	Summary          string              `json:"summary,omitempty"`
	Description      string              `json:"description,omitempty"`
	Parameters       []Parameter         `json:"parameters,omitempty"`
	ResponseMessages []ResponseMessage   `json:"responseMessages,omitempty"` // optional
	Consumes         []string            `json:"consumes,omitempty"`
	Produces         []string            `json:"produces,omitempty"`
	Authorizations   []Authorization     `json:"authorizations,omitempty"`
	Responses        map[string]Response `json:"responses"`
	Tags             []string            `json:"tags,omitempty"`
	IsMonitoring     bool                `json:"-"`
	Deprecated       bool                `json:"deprecated"`
}

//NewOperation returns an op
func NewOperation(httpMethod, nickname, summary, typ, description string, deprecated bool) (op Operation) {

	op.HttpMethod = httpMethod
	op.Nickname = nickname
	op.Summary = summary
	op.Description = description
	op.Summary = summary
	op.Deprecated = deprecated
	op.Type = typ
	if op.Type == "" {
		op.Type = "void"
	}
	return
}

//AddParameter adds a param to operation
func (op *Operation) AddParameter(param Parameter) {

	params := make([]Parameter, len(op.Parameters)+1)

	for i := range op.Parameters {
		params[i] = op.Parameters[i]
	}
	params[len(op.Parameters)] = param

	op.Parameters = params
}
