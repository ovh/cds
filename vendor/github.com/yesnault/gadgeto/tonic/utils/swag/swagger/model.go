package swagger

//Model is https://github.com/swagger-api/swagger-spec/blob/master/versions/1.2.md#527-model-object
type Model struct {
	Id          string                    `json:"-"`
	Description string                    `json:"description,omitempty"`
	Required    []string                  `json:"required,omitempty"`
	Properties  map[string]*ModelProperty `json:"properties"`
}

// TODO Unify this struct and swagger.Schema in a common struct to accomodate model, params, response types
type NestedItems struct {
	Type                 string       `json:"type,omitempty"`
	RefId                string       `json:"$ref,omitempty"`
	Items                *NestedItems `json:"items,omitempty"`
	AdditionalProperties *NestedItems `json:"additionalProperties,omitempty"`
}

// ModelProperty is https://github.com/swagger-api/swagger-spec/blob/master/versions/1.2.md#528-properties-object
type ModelProperty struct {
	Type                 string       `json:"type,omitempty"`
	RefId                string       `json:"$ref,omitempty"`
	Format               string       `json:"format,omitempty"`
	Description          string       `json:"description,omitempty"`
	Items                *NestedItems `json:"items,omitempty"`
	AdditionalProperties *NestedItems `json:"additionalProperties,omitempty"`
	Enum                 []string     `json:"enum,omitempty"`
	Required             bool         `json:"-"`
	HideOnListing        bool         `json:"-"` //Hide this property on listing methods, activated with the swag omitinlisting
}

//NewModel bootstraps a model ...
func NewModel(id string) Model {
	m := Model{Id: id}
	m.Required = []string{}
	m.Properties = map[string]*ModelProperty{}

	return m
}
