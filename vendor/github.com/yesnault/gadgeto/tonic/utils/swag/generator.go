package swag

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/loopfz/gadgeto/tonic/utils/swag/doc"
	"github.com/loopfz/gadgeto/tonic/utils/swag/swagger"
)

// GENERATOR

// SchemaGenerator is the object users have to manipulate, it internally collects data about packages used by handlers
// and so on
type SchemaGenerator struct {
	apiDeclaration *swagger.ApiDeclaration

	swaggedTypesMap map[reflect.Type]*swagger.Model
	generatedTypes  map[reflect.Type]string

	genesisDefer map[reflect.Type][]nameSetter
	genesis      map[reflect.Type]bool

	docInfos *doc.Infos
}

// NewSchemaGenerator bootstraps a generator, don't instantiate SchemaGenerator yourself
func NewSchemaGenerator() *SchemaGenerator {
	s := &SchemaGenerator{}
	s.swaggedTypesMap = map[reflect.Type]*swagger.Model{}
	s.generatedTypes = map[reflect.Type]string{}
	s.genesisDefer = map[reflect.Type][]nameSetter{}
	s.genesis = map[reflect.Type]bool{}

	return s
}

// GenerateSwagDeclaration parses all routes (handlers, structs) and returns ready to serialize/use ApiDeclaration
func (s *SchemaGenerator) GenerateSwagDeclaration(routes map[string]*tonic.Route, basePath, version string, godoc *doc.Infos) error {

	s.docInfos = godoc
	s.apiDeclaration = swagger.NewApiDeclaration(version, basePath)

	// create Operation for each route, creating models as we go
	for _, route := range routes {
		if err := s.addOperation(route); err != nil {
			return err
		}
	}

	for t, list := range s.genesisDefer {
		for _, ns := range list {
			if ns == nil {
				if reflect.Ptr != t.Kind() {
					//fmt.Println("incomplete generator: missing defered setter somewhere. FYI type was: " + t.Name() + " / " + t.Kind().String())
				}
			} else {
				ns(s.generatedTypes[t])
			}
		}
	}

	return nil
}

func (s *SchemaGenerator) generateModels(routes map[string]*tonic.Route) error {
	for _, route := range routes {
		s.generateSwagModel(route.GetInType(), nil)
		s.generateSwagModel(route.GetOutType(), nil)
	}

	return nil
}

func (s *SchemaGenerator) addOperation(route *tonic.Route) error {

	op, err := s.generateOperation(route)
	if err != nil {
		return err
	}

	path := cleanPath(route.GetPath())
	if _, ok := s.apiDeclaration.Paths[path]; !ok {
		s.apiDeclaration.Paths[path] = make(map[string]swagger.Operation)
	}
	s.apiDeclaration.Paths[path][strings.ToLower(op.HttpMethod)] = *op

	return nil
}

func (s *SchemaGenerator) generateOperation(route *tonic.Route) (*swagger.Operation, error) {

	in := route.GetInType()
	out := route.GetOutType()

	desc := s.docInfos.FunctionsDoc[route.GetHandlerNameWithPackage()]
	if desc == "" {
		desc = route.GetDescription()
	}

	op := swagger.NewOperation(
		route.GetVerb(),
		route.GetHandlerName(),
		route.GetSummary(),
		s.generateSwagModel(out, nil),
		desc,
		route.GetDeprecated(),
	)

	if err := s.setOperationParams(&op, in); err != nil {
		return nil, err
	}
	if err := s.setOperationResponse(&op, out, route.GetDefaultStatusCode()); err != nil {
		return nil, err
	}

	op.Tags = route.GetTags()

	return &op, nil
}

// sometimes recursive types can only be fully determined
// after full analysis, we use this interface to do so
type nameSetter func(string)

// ###################################

func (s *SchemaGenerator) setOperationResponse(op *swagger.Operation, t reflect.Type, retcode int) error {

	//Just give every method a 200 response for now
	//This could be improved given for example 201 for post
	//methods etc
	schema := swagger.Schema{}
	schemaType := s.generateSwagModel(t, nil)
	if strings.Contains(schemaType, "#/") {
		if t.Kind() == reflect.Slice {
			schema.Type = "array"
			if schema.Items == nil {
				schema.Items = make(map[string]string)
			}
			schema.Items["$ref"] = schemaType
		} else {
			schema.Ref = schemaType
		}
	} else {
		schema.Type = schemaType
	}

	response := swagger.Response{}
	if schema.Type != "void" {
		//For void params swagger def can't be "void"
		//No schema at all is fine
		response.Schema = &schema
	}

	op.Responses = map[string]swagger.Response{
		fmt.Sprintf("%d", retcode): response,
	}

	return nil
}

func (s *SchemaGenerator) setOperationParams(op *swagger.Operation, in reflect.Type) error {

	if in == nil || in.Kind() != reflect.Struct {
		return nil
	}

	var body *swagger.Model

	for i := 0; i < in.NumField(); i++ {
		// Embedded field found, extract its fields
		// as top-level parameters.
		if in.Field(i).Anonymous {
			for y := 0; y < in.Field(i).Type.NumField(); y++ {
				p := s.newParamFromStructField(in.Field(i).Type.Field(y), &body)
				if p != nil {
					if doc := s.docInfos.StructFieldsDoc[in.Name()]; doc != nil {
						if fieldDoc := doc[in.Field(i).Name]; fieldDoc != "" {
							p.Description = strings.TrimSuffix(fieldDoc, "\n")
						}
					}
					op.AddParameter(*p)
				}
			}
		} else {
			p := s.newParamFromStructField(in.Field(i), &body)
			if p != nil {
				if doc := s.docInfos.StructFieldsDoc[in.Name()]; doc != nil {
					if fieldDoc := doc[in.Field(i).Name]; fieldDoc != "" {
						p.Description = strings.TrimSuffix(fieldDoc, "\n")
					}
				}
				op.AddParameter(*p)
			}
		}
	}
	if body != nil {
		body.Id = "Input" + op.Nickname + "In"
		s.apiDeclaration.AddModel(*body)
		bodyParam := swagger.NewParameter("body", "body", "body request", true, false, "", "", "")
		bodyParam.Schema.Ref = "#/definitions/" + body.Id
		op.AddParameter(bodyParam)
	}

	return nil

}

func (s *SchemaGenerator) newParamFromStructField(f reflect.StructField, bodyModel **swagger.Model) *swagger.Parameter {

	s.generateSwagModel(f.Type, nil)

	name := paramName(f)
	paramType := paramType(f)

	if paramType == "body" {
		if *bodyModel == nil {
			m := swagger.NewModel("Input")
			*bodyModel = &m
		}
		(*bodyModel).Properties[name] = s.fieldToModelProperty(f)
		return nil
	}

	_, allowMultiple := paramTargetTypeAllowMultiple(f)
	format, dataType, refId := paramFormatDataTypeRefId(f)

	p := swagger.NewParameter(
		paramType,
		name,
		paramDescription(f),
		paramRequired(f),
		allowMultiple,
		dataType,
		format,
		refId,
	)

	if tag := f.Tag.Get("enum"); tag != "" {
		p.Enum = strings.Split(tag, ",")
	}
	p.Default = paramsDefault(f)

	// extra swagger specific tags.
	p.CollectionFormat = f.Tag.Get("swagger-collection-format")

	return &p
}

func (s *SchemaGenerator) generateSwagModel(t reflect.Type, ns nameSetter) string {
	// nothing to generate
	if t == nil {
		s.generatedTypes[t] = "void"
		return "void"
	}

	//Check if we alredy seen this type
	if finalName, ok := s.generatedTypes[t]; ok {
		return finalName
	}

	if s.genesis[t] {
		if s.genesisDefer[t] == nil {
			s.genesisDefer[t] = []nameSetter{ns}
		} else {
			s.genesisDefer[t] = append(s.genesisDefer[t], ns)
		}
		return "defered"
	}

	s.genesis[t] = true
	defer func() { s.genesis[t] = false }()

	if "Time" == t.Name() && t.PkgPath() == "time" {
		s.generatedTypes[t] = "dateTime (sdk borken)"
		return "dateTime (sdk borken)" //TODO: this is wrong, if a function has to return time we would need type + format
	}

	// let's treat pointed type
	if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return s.generateSwagModel(t.Elem(), ns)
	}

	// we have either a simple type or a constant/aliased one
	if t.Kind() != reflect.Struct {
		if t.Kind().String() != t.Name() {
			return t.Kind().String()
		}
		typeName, _, _ := swagger.GoTypeToSwagger(t)
		s.generatedTypes[t] = typeName
		return typeName
	}

	modelName := swagger.ModelName(t)
	if modelName == "" { // Bozo : I can't find the guilty type coming in, probably wosk.Null but ...
		s.generatedTypes[t] = "void"
		return "void"
	}

	if _, ok := s.swaggedTypesMap[t]; ok {
		s.generatedTypes[t] = modelName
		return modelName
	}

	m := swagger.NewModel(modelName)
	if t.Kind() == reflect.Struct {
		structFields := s.getStructFields(t, ns)
		if len(structFields) > 0 {
			for name, property := range structFields {
				m.Properties[name] = property
			}
			s.apiDeclaration.AddModel(m)
		}
	}

	s.swaggedTypesMap[t] = &m
	s.generatedTypes[t] = "#/definitions/" + m.Id

	return "#/definitions/" + m.Id
}

func (s *SchemaGenerator) getStructFields(t reflect.Type, ns nameSetter) map[string]*swagger.ModelProperty {
	structFields := make(map[string]*swagger.ModelProperty)

	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type.Kind() == reflect.Func {
			//Ignore functions
			continue
		}

		name := getFieldName(t.Field(i))
		if name == nil {
			continue
		}

		if t.Field(i).Anonymous {
			//For anonymous (embedded) fields, we flatten their structure, ie, we add the fields
			//to the parent model.
			typeToUse := t.Field(i).Type
			if typeToUse.Kind() == reflect.Ptr {
				typeToUse = t.Field(i).Type.Elem()
			}
			dbModelFields := s.getStructFields(typeToUse, ns)
			for fieldName, property := range dbModelFields {
				structFields[fieldName] = property
			}

		} else {
			// for fields that are not of simple types, we "program" generation
			s.generateSwagModel(t.Field(i).Type, ns)
			property := s.fieldToModelProperty(t.Field(i))
			structFields[*name] = property
		}

	}

	return structFields
}

func (s *SchemaGenerator) getNestedItemType(t reflect.Type, p *swagger.ModelProperty) *swagger.NestedItems {
	arrayItems := &swagger.NestedItems{}

	if t.Kind() == reflect.Struct {
		arrayItems.RefId = s.generateSwagModel(t, func(a string) { arrayItems.RefId = a })
	} else if t.Kind() == reflect.Map {
		arrayItems.AdditionalProperties = s.getNestedItemType(t.Elem(), p)
		arrayItems.Type = "object"
	} else if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		arrayItems.Items = s.getNestedItemType(t.Elem(), p)
		arrayItems.Type = "array"
	} else {
		arrayItems.Type = s.generateSwagModel(t, nil)
	}
	return arrayItems
}

// Turns a field of a struct to a model property
func (s *SchemaGenerator) fieldToModelProperty(f reflect.StructField) *swagger.ModelProperty {
	// TODO we should know whether struct is inbound or outbound
	p := &swagger.ModelProperty{Required: true}
	if f.Tag.Get("wosk") != "" {
		if strings.Index(f.Tag.Get("wosk"), "required=false") != -1 {
			p.Required = false
		}
	}

	if f.Tag.Get("description") != "" {
		p.Description = f.Tag.Get("description")
	}

	if f.Tag.Get("swagger-type") != "" {
		//Swagger type defined on the original struct, no need to infer it
		//format is: swagger-type:type[,format]
		tagValue := f.Tag.Get("swagger-type")
		tagTypes := strings.Split(tagValue, ",")
		switch len(tagTypes) {
		case 1:
			p.Type = tagTypes[0]
		case 2:
			p.Type = tagTypes[0]
			p.Format = tagTypes[1]
		default:
			panic(fmt.Sprintf("Error: bad swagger-type definition on %s (%s)", f.Name, tagValue))
		}
	} else {

		if f.Type.Kind() == reflect.Slice || f.Type.Kind() == reflect.Array {
			p.Type = "array"
			targetType := f.Type.Elem()
			if targetType.Kind() == reflect.Ptr {
				targetType = targetType.Elem()
			}
			if targetType.Kind() == reflect.Map {
				p.Items = &swagger.NestedItems{}
				nestedItem := s.getNestedItemType(targetType.Elem(), p)
				p.Items.AdditionalProperties = nestedItem
				p.Items.Type = "object"

			} else {
				p.Items = s.getNestedItemType(targetType, p)
			}
		} else if f.Type.Kind() == reflect.Map {
			if f.Type.Key().Kind() != reflect.String {
				fmt.Fprintln(os.Stderr, "Type not supported, only map with string keys, got: ", f.Type.Key())
			}
			p.Type = "object"
			p.AdditionalProperties = &swagger.NestedItems{}
			targetType := f.Type.Elem()
			if targetType.Kind() == reflect.Ptr {
				targetType = targetType.Elem()
			}
			typ := s.generateSwagModel(targetType, nil)
			if targetType.Kind() == reflect.Struct {
				p.AdditionalProperties.RefId = typ
			} else if targetType.Kind() == reflect.Slice || targetType.Kind() == reflect.Array {
				nestedItem := s.getNestedItemType(targetType.Elem(), p)
				p.AdditionalProperties.Items = nestedItem
				p.AdditionalProperties.Type = "array"
			} else {
				p.AdditionalProperties.Type = typ
			}

		} else {
			t := f.Type
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			if "Time" == t.Name() && t.PkgPath() == "time" {
				p.Type = "string"
				p.Format = "dateTime"
			} else if t.Kind() == reflect.Struct {
				p.RefId = s.generateSwagModel(t, func(a string) { p.RefId = a })
			} else { // if it's a constant, maybe it's an enum
				s.generateSwagModel(t, nil)
				if list, ok := s.docInfos.Constants[t.String()]; ok {
					values := []string{}
					for _, co := range list.ListC {
						// TODO this is WRONG !
						// TODO I only copy names of constants, we'd need to actually evaluate value
						// I can only think of generating a script, go run it to get values :(
						values = append(values, co)
					}
					p.Enum = values
					p.Description = "WARNING: constants are constants names, they should be values (swagger generator incomplete)"
				}
				p.Type = s.generateSwagModel(t, func(n string) { p.Type = n })
			}
		}
	}
	return p
}
