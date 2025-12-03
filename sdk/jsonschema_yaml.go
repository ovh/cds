package sdk

import (
	"fmt"
	"io"
	"strings"

	"github.com/sguiheux/jsonschema"
)

// YAMLGenerator generate a documented YAML output from a JSONSchema
type YAMLGenerator struct {
	indent      string
	commentChar string
}

// NewYAMLGenerator creates a generator with default configuration
func NewYAMLGenerator() *YAMLGenerator {
	return &YAMLGenerator{
		indent:      "  ",
		commentChar: "#",
	}
}

// Generate writes the YAML example to the provided writer
func (g *YAMLGenerator) Generate(w io.Writer, schema *jsonschema.Schema) error {
	// Resolve root reference if present
	resolved := g.resolveRef(schema, schema)
	return g.generateSchema(w, resolved, schema, 0, "")
}

func (g *YAMLGenerator) generateSchema(w io.Writer, schema *jsonschema.Schema, root *jsonschema.Schema, level int, parentKey string) error {
	if schema == nil {
		return nil
	}

	// Handle references
	if schema.Ref != "" {
		resolved := g.resolveRef(schema, root)
		if resolved != nil && resolved != schema {
			return g.generateSchema(w, resolved, root, level, parentKey)
		}
		// If not resolved, display a comment
		fmt.Fprintf(w, "%s%s ref: %s\n", g.indentStr(level), g.commentChar, schema.Ref)
		return nil
	}

	// Note: description is now handled in generateObject on the key line
	// We don't display it here to avoid duplicates

	switch schema.Type {
	case "object":
		return g.generateObject(w, schema, root, level, parentKey)
	case "array":
		return g.generateArray(w, schema, root, level, parentKey)
	case "string":
		g.generateString(w, schema, level, parentKey)
	case "number", "integer":
		g.generateNumber(w, schema, level, parentKey)
	case "boolean":
		g.generateBoolean(w, schema, level, parentKey)
	default:
		// Unspecified type or anyOf/oneOf
		if len(schema.AnyOf) > 0 || len(schema.OneOf) > 0 {
			g.generateUnion(w, schema, root, level, parentKey)
		} else {
			fmt.Fprintf(w, "%s%s\n", g.indentStr(level), g.exampleValue(schema))
		}
	}

	return nil
}

// resolveRef resolves a $ref reference in the schema
func (g *YAMLGenerator) resolveRef(schema *jsonschema.Schema, root *jsonschema.Schema) *jsonschema.Schema {
	if schema.Ref == "" {
		return schema
	}

	// Expected format: #/$defs/TypeName or #/definitions/TypeName
	ref := schema.Ref
	if !strings.HasPrefix(ref, "#/") {
		return schema
	}

	// Extract path after #/
	path := strings.TrimPrefix(ref, "#/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		return schema
	}

	// Search in definitions
	if parts[0] == "$defs" || parts[0] == "definitions" {
		// First, try in the root definitions
		if root.Definitions != nil {
			if defInterface, ok := root.Definitions[parts[1]]; ok {
				def := g.toSchemaPtr(defInterface)
				if def != nil {
					// If the resolved schema itself has a $ref, resolve it from its own $defs
					if def.Ref != "" && def.Definitions != nil {
						return g.resolveRef(def, def)
					}
					return def
				}
			}
		}

		// If not found in root, try in schema's own definitions
		if schema.Definitions != nil {
			if defInterface, ok := schema.Definitions[parts[1]]; ok {
				def := g.toSchemaPtr(defInterface)
				if def != nil {
					return def
				}
			}
		}
	}

	return schema
}

func (g *YAMLGenerator) generateObject(w io.Writer, schema *jsonschema.Schema, root *jsonschema.Schema, level int, parentKey string) error {
	// Check for AllOf with if/then conditions (for worker model spec variants)
	if len(schema.AllOf) > 0 {
		return g.generateAllOfVariants(w, schema, root, level, parentKey)
	}

	// Special case: map with PatternProperties (dynamic keys)
	if (schema.Properties == nil || len(schema.Properties.Keys()) == 0) && len(schema.PatternProperties) > 0 {
		return g.generateMap(w, schema, root, level, parentKey)
	}

	if schema.Properties == nil {
		fmt.Fprintf(w, "%s{}\n", g.indentStr(level))
		return nil
	}

	keys := schema.Properties.Keys()
	if len(keys) == 0 {
		fmt.Fprintf(w, "%s{}\n", g.indentStr(level))
		return nil
	}

	for i, key := range keys {
		propInterface, ok := schema.Properties.Get(key)
		if !ok {
			continue
		}
		prop := propInterface.(*jsonschema.Schema)

		// Resolve $ref references in the property
		if prop.Ref != "" {
			prop = g.resolveRef(prop, root)
		}

		// Visual separator between properties (except for the first)
		if i > 0 && (prop.Type == "object" || prop.Type == "array") {
			fmt.Fprintln(w)
		}

		hasProps := prop.Properties != nil && len(prop.Properties.Keys()) > 0
		hasPatternProps := len(prop.PatternProperties) > 0
		hasUnion := len(prop.AnyOf) > 0 || len(prop.OneOf) > 0

		// Build comment with description
		comment := ""
		if prop.Description != "" {
			comment = " # " + prop.Description
		}

		// For objects and arrays: key: # description
		if prop.Type == "object" && (hasProps || hasPatternProps) {
			fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), key, comment)
			g.generateSchema(w, prop, root, level+1, key)
		} else if prop.Type == "array" {
			// Check if it's an array of scalars
			if prop.Items != nil {
				items := prop.Items
				if items.Ref != "" {
					items = g.resolveRef(items, root)
				}
				hasItemProps := items.Properties != nil && len(items.Properties.Keys()) > 0
				hasItemPatternProps := len(items.PatternProperties) > 0

				// If scalar array: key: []
				if items.Type != "object" || (!hasItemProps && !hasItemPatternProps) {
					fmt.Fprintf(w, "%s%s: []%s\n", g.indentStr(level), key, comment)
				} else {
					// Object array: display the structure
					fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), key, comment)
					g.generateSchema(w, prop, root, level+1, key)
				}
			} else {
				fmt.Fprintf(w, "%s%s: []%s\n", g.indentStr(level), key, comment)
			}
		} else if hasUnion {
			// Check if union contains only scalar types
			variants := prop.AnyOf
			if len(variants) == 0 {
				variants = prop.OneOf
			}

			isScalarUnion := true
			for _, variant := range variants {
				vType := variant.Type
				if vType == "object" || vType == "array" {
					isScalarUnion = false
					break
				}
			}

			if isScalarUnion && len(variants) > 0 {
				// For scalar unions: display first variant value on the same line
				firstVariant := variants[0]
				if firstVariant.Ref != "" {
					firstVariant = g.resolveRef(firstVariant, root)
				}
				value := g.exampleValue(firstVariant)
				fmt.Fprintf(w, "%s%s: %s%s\n", g.indentStr(level), key, value, comment)
			} else {
				// For complex unions: display on separate lines
				fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), key, comment)
				g.generateSchema(w, prop, root, level+1, key)
			}
		} else if len(prop.AllOf) > 0 {
			// AllOf: generate all variants (for conditional schemas like worker model spec)
			fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), key, comment)
			g.generateAllOf(w, prop, root, level+1, key)
		} else {
			// For scalars: key: <value> # description
			value := g.exampleValue(prop)
			fmt.Fprintf(w, "%s%s: %s%s\n", g.indentStr(level), key, value, comment)
		}
	}

	return nil
}

func (g *YAMLGenerator) generateMap(w io.Writer, schema *jsonschema.Schema, root *jsonschema.Schema, level int, parentKey string) error {
	// Map with PatternProperties - generate an example key-value pair
	for pattern, valueSchema := range schema.PatternProperties {
		// Generate an example key based on pattern and parent
		exampleKey := g.exampleKeyFromPattern(pattern, parentKey)

		// Resolve references before checking properties
		if valueSchema.Ref != "" {
			valueSchema = g.resolveRef(valueSchema, root)
		}

		// Generate the example value
		hasProps := valueSchema.Properties != nil && len(valueSchema.Properties.Keys()) > 0

		// If schema has properties, it's an object even if Type is not explicitly "object"
		if hasProps {
			// Object map: key: # description then indented properties
			comment := ""
			if valueSchema.Description != "" {
				comment = " # " + valueSchema.Description
			}
			fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), exampleKey, comment)
			g.generateObject(w, valueSchema, root, level+1, exampleKey)
		} else {
			// Scalar map: key: <value> # description
			value := g.exampleValue(valueSchema)
			comment := ""
			if valueSchema.Description != "" {
				comment = " # " + valueSchema.Description
			}
			fmt.Fprintf(w, "%s%s: %s%s\n", g.indentStr(level), exampleKey, value, comment)
		}

		break // Only one pattern for the example
	}

	return nil
}

func (g *YAMLGenerator) generateArray(w io.Writer, schema *jsonschema.Schema, root *jsonschema.Schema, level int, parentKey string) error {
	if schema.Items == nil {
		fmt.Fprintf(w, "%s[]\n", g.indentStr(level))
		return nil
	}

	items := schema.Items

	// Resolve references in items
	if items.Ref != "" {
		items = g.resolveRef(items, root)
	}

	// Complex object array
	hasProps := items.Properties != nil && len(items.Properties.Keys()) > 0
	hasPatternProps := len(items.PatternProperties) > 0

	if items.Type == "object" && (hasProps || hasPatternProps) {
		fmt.Fprintf(w, "%s-\n", g.indentStr(level))
		if hasProps {
			g.generateSchema(w, items, root, level+1, parentKey)
		} else if hasPatternProps {
			g.generateMap(w, items, root, level+1, parentKey)
		}
		return nil
	}

	// For scalars: - <value> # description
	value := g.exampleValue(items)
	comment := ""
	if items.Description != "" {
		comment = " # " + items.Description
	}
	fmt.Fprintf(w, "%s- %s%s\n", g.indentStr(level), value, comment)
	return nil
}

func (g *YAMLGenerator) generateString(w io.Writer, schema *jsonschema.Schema, level int, parentKey string) {
	// <value> (no comment, as it's handled by parent on the key line)
	value := g.exampleValue(schema)
	if level > 0 {
		fmt.Fprintf(w, "%s%s\n", g.indentStr(level), value)
	} else {
		fmt.Fprintf(w, "%s\n", value)
	}
}

func (g *YAMLGenerator) generateNumber(w io.Writer, schema *jsonschema.Schema, level int, parentKey string) {
	// <value> (no comment, as it's handled by parent on the key line)
	value := g.exampleValue(schema)
	if level > 0 {
		fmt.Fprintf(w, "%s%s\n", g.indentStr(level), value)
	} else {
		fmt.Fprintf(w, "%s\n", value)
	}
}

func (g *YAMLGenerator) generateBoolean(w io.Writer, schema *jsonschema.Schema, level int, parentKey string) {
	// <value> (no comment, as it's handled by parent on the key line)
	value := g.exampleValue(schema)
	if level > 0 {
		fmt.Fprintf(w, "%s%s\n", g.indentStr(level), value)
	} else {
		fmt.Fprintf(w, "%s\n", value)
	}
}

func (g *YAMLGenerator) generateUnion(w io.Writer, schema *jsonschema.Schema, root *jsonschema.Schema, level int, parentKey string) {
	variants := schema.AnyOf
	if len(variants) == 0 {
		variants = schema.OneOf
	}

	if len(variants) > 0 {
		// Generate all variants with their complete properties
		for i, variant := range variants {
			if variant.Ref != "" {
				variant = g.resolveRef(variant, root)
			}

			// If variant is an object with properties, display them
			hasProps := variant.Properties != nil && len(variant.Properties.Keys()) > 0
			hasPatternProps := len(variant.PatternProperties) > 0

			if variant.Type == "object" && (hasProps || hasPatternProps) {
				if i > 0 {
					fmt.Fprintln(w)
				}
				g.generateSchema(w, variant, root, level, parentKey)
			} else if variant.Type == "array" {
				if i > 0 {
					fmt.Fprintln(w)
				}
				g.generateSchema(w, variant, root, level, parentKey)
			} else {
				// For scalars in union: just <value> (no comment as it's already on the key line)
				value := g.exampleValue(variant)
				fmt.Fprintf(w, "%s%s\n", g.indentStr(level), value)
			}
		}
	}
}

// generateAllOfVariants handles AllOf schemas with if/then conditions
// Used for worker model spec variants (docker, openstack, vsphere)
func (g *YAMLGenerator) generateAllOfVariants(w io.Writer, schema *jsonschema.Schema, root *jsonschema.Schema, level int, parentKey string) error {
	// First, generate the base properties (without spec)
	if schema.Properties != nil {
		keys := schema.Properties.Keys()
		for i, key := range keys {
			propInterface, ok := schema.Properties.Get(key)
			if !ok {
				continue
			}
			prop := g.toSchemaPtr(propInterface)
			if prop == nil {
				continue
			}

			// Resolve $ref references in the property
			if prop.Ref != "" {
				prop = g.resolveRef(prop, root)
			}

			// Visual separator between properties (except for the first)
			if i > 0 && (prop.Type == "object" || prop.Type == "array") {
				fmt.Fprintln(w)
			}

			hasProps := prop.Properties != nil && len(prop.Properties.Keys()) > 0
			hasPatternProps := len(prop.PatternProperties) > 0
			hasUnion := len(prop.AnyOf) > 0 || len(prop.OneOf) > 0

			// Build comment with description
			comment := ""
			if prop.Description != "" {
				comment = " # " + prop.Description
			}

			// Skip properties that will be handled by AllOf (like spec)
			skipProperty := false
			for _, allOfItem := range schema.AllOf {
				if allOfItem.Then != nil && allOfItem.Then.Properties != nil {
					if _, hasKey := allOfItem.Then.Properties.Get(key); hasKey {
						skipProperty = true
						break
					}
				}
			}

			if skipProperty {
				// For spec, we'll generate all variants
				fmt.Fprintf(w, "\n")
				fmt.Fprintf(w, "%s# Examples with different types:\n", g.indentStr(level))

				// Generate each variant based on if/then conditions
				for variantIdx, allOfItem := range schema.AllOf {
					if allOfItem.If != nil && allOfItem.Then != nil {
						// Extract the condition value (e.g., docker, openstack, vsphere)
						condValue := g.extractIfConditionValue(allOfItem.If)
						if condValue != "" {
							fmt.Fprintf(w, "\n")
							fmt.Fprintf(w, "%s# Example %d: type=%s\n", g.indentStr(level), variantIdx+1, condValue)

							// Generate all properties for this variant
							for _, vkey := range keys {
								vpropInterface, _ := schema.Properties.Get(vkey)
								vprop := g.toSchemaPtr(vpropInterface)
								if vprop == nil {
									continue
								}
								if vprop.Ref != "" {
									vprop = g.resolveRef(vprop, root)
								}

								vcomment := ""
								if vprop.Description != "" {
									vcomment = " # " + vprop.Description
								}

								// For the discriminator property (type), use the condition value
								if allOfItem.If.Properties != nil {
									if _, isDiscriminator := allOfItem.If.Properties.Get(vkey); isDiscriminator {
										fmt.Fprintf(w, "%s%s: %s%s\n", g.indentStr(level), vkey, condValue, vcomment)
										continue
									}
								}

								// For spec, use the variant from Then
								if allOfItem.Then.Properties != nil {
									if variantPropInterface, hasVariantKey := allOfItem.Then.Properties.Get(vkey); hasVariantKey {
										variantProp := g.toSchemaPtr(variantPropInterface)
										if variantProp == nil {
											continue
										}
										// Resolve refs iteratively (handle nested refs)
										maxIterations := 10
										for i := 0; i < maxIterations && variantProp.Ref != ""; i++ {
											resolved := g.resolveRef(variantProp, root)
											if resolved == variantProp || resolved == nil {
												break
											}
											variantProp = resolved
										}

										hasVProps := variantProp.Properties != nil && len(variantProp.Properties.Keys()) > 0
										hasVPatternProps := len(variantProp.PatternProperties) > 0

										if hasVProps || hasVPatternProps {
											fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), vkey, vcomment)
											g.generateSchema(w, variantProp, root, level+1, vkey)
										} else {
											value := g.exampleValue(variantProp)
											fmt.Fprintf(w, "%s%s: %s%s\n", g.indentStr(level), vkey, value, vcomment)
										}
										continue
									}
								}

								// For other properties, generate normally
								vhasProps := vprop.Properties != nil && len(vprop.Properties.Keys()) > 0
								vhasPatternProps := len(vprop.PatternProperties) > 0
								vhasUnion := len(vprop.AnyOf) > 0 || len(vprop.OneOf) > 0

								if vprop.Type == "object" && (vhasProps || vhasPatternProps) {
									fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), vkey, vcomment)
									g.generateSchema(w, vprop, root, level+1, vkey)
								} else if vprop.Type == "array" {
									if vprop.Items != nil {
										items := vprop.Items
										if items.Ref != "" {
											items = g.resolveRef(items, root)
										}
										hasItemProps := items.Properties != nil && len(items.Properties.Keys()) > 0
										hasItemPatternProps := len(items.PatternProperties) > 0

										if items.Type != "object" || (!hasItemProps && !hasItemPatternProps) {
											fmt.Fprintf(w, "%s%s: []%s\n", g.indentStr(level), vkey, vcomment)
										} else {
											fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), vkey, vcomment)
											g.generateSchema(w, vprop, root, level+1, vkey)
										}
									} else {
										fmt.Fprintf(w, "%s%s: []%s\n", g.indentStr(level), vkey, vcomment)
									}
								} else if vhasUnion {
									fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), vkey, vcomment)
									g.generateSchema(w, vprop, root, level+1, vkey)
								} else {
									value := g.exampleValue(vprop)
									fmt.Fprintf(w, "%s%s: %s%s\n", g.indentStr(level), vkey, value, vcomment)
								}
							}
						}
					}
				}
				continue
			}

			// For objects and arrays: key: # description
			if prop.Type == "object" && (hasProps || hasPatternProps) {
				fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), key, comment)
				g.generateSchema(w, prop, root, level+1, key)
			} else if prop.Type == "array" {
				// Check if it's an array of scalars
				if prop.Items != nil {
					items := prop.Items
					if items.Ref != "" {
						items = g.resolveRef(items, root)
					}
					hasItemProps := items.Properties != nil && len(items.Properties.Keys()) > 0
					hasItemPatternProps := len(items.PatternProperties) > 0

					// If scalar array: key: []
					if items.Type != "object" || (!hasItemProps && !hasItemPatternProps) {
						fmt.Fprintf(w, "%s%s: []%s\n", g.indentStr(level), key, comment)
					} else {
						// Object array: display the structure
						fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), key, comment)
						g.generateSchema(w, prop, root, level+1, key)
					}
				} else {
					fmt.Fprintf(w, "%s%s: []%s\n", g.indentStr(level), key, comment)
				}
			} else if hasUnion {
				fmt.Fprintf(w, "%s%s:%s\n", g.indentStr(level), key, comment)
				g.generateSchema(w, prop, root, level+1, key)
			} else {
				// For scalars: key: <value> # description
				value := g.exampleValue(prop)
				fmt.Fprintf(w, "%s%s: %s%s\n", g.indentStr(level), key, value, comment)
			}
		}
	}

	return nil
}

// toSchemaPtr converts interface{} to *jsonschema.Schema handling both value and pointer types
func (g *YAMLGenerator) toSchemaPtr(v interface{}) *jsonschema.Schema {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case *jsonschema.Schema:
		return s
	case jsonschema.Schema:
		return &s
	default:
		return nil
	}
}

// extractIfConditionValue extracts the constant value from an if condition
// Used to get "docker", "openstack", or "vsphere" from if/then conditions
func (g *YAMLGenerator) extractIfConditionValue(ifSchema *jsonschema.Schema) string {
	if ifSchema == nil || ifSchema.Properties == nil {
		return ""
	}

	// Look for a property with a const value
	keys := ifSchema.Properties.Keys()
	for _, key := range keys {
		propInterface, ok := ifSchema.Properties.Get(key)
		if !ok {
			continue
		}

		prop := g.toSchemaPtr(propInterface)
		if prop == nil {
			continue
		}

		if prop.Const != nil {
			return fmt.Sprintf("%v", prop.Const)
		}
	}
	return ""
}

func (g *YAMLGenerator) generateAllOf(w io.Writer, schema *jsonschema.Schema, root *jsonschema.Schema, level int, parentKey string) {
	if len(schema.AllOf) == 0 {
		return
	}

	// Generate all AllOf variants
	for i, variant := range schema.AllOf {
		if variant.Ref != "" {
			variant = g.resolveRef(variant, root)
		}

		// If variant is an object with properties, display them
		hasProps := variant.Properties != nil && len(variant.Properties.Keys()) > 0
		hasPatternProps := len(variant.PatternProperties) > 0

		if variant.Type == "object" && (hasProps || hasPatternProps) {
			if i > 0 {
				fmt.Fprintln(w)
			}
			g.generateSchema(w, variant, root, level, parentKey)
		} else if variant.Type == "array" {
			if i > 0 {
				fmt.Fprintln(w)
			}
			g.generateSchema(w, variant, root, level, parentKey)
		} else {
			// For scalars: just <value>
			value := g.exampleValue(variant)
			fmt.Fprintf(w, "%s%s\n", g.indentStr(level), value)
		}
	}
}

// exampleValue generates an example value based on type and constraints
// Don't include description here, it will be in the comment
func (g *YAMLGenerator) exampleValue(schema *jsonschema.Schema) string {
	// Default: use default value
	if schema.Default != nil {
		return fmt.Sprintf("%v", schema.Default)
	}

	// Example: use provided example
	if len(schema.Examples) > 0 {
		return fmt.Sprintf("%v", schema.Examples[0])
	}

	// If no explicit type but has enum or pattern, it's likely a string
	inferredType := schema.Type
	if inferredType == "" {
		if len(schema.Enum) > 0 || schema.Pattern != "" {
			inferredType = "string"
		}
	}

	// According to type
	switch inferredType {
	case "string":
		if schema.Pattern != "" {
			return fmt.Sprintf("<string matching: %s>", schema.Pattern)
		}
		return "<string>"
	case "number":
		if schema.Minimum != 0 {
			return fmt.Sprintf("%d.0", schema.Minimum)
		}
		return "<number>"
	case "integer":
		if schema.Minimum != 0 {
			return fmt.Sprintf("%d", schema.Minimum)
		}
		return "<integer>"
	case "boolean":
		return "<boolean>"
	case "null":
		return "null"
	case "array":
		return "[]"
	case "object":
		return "{}"
	default:
		return "<value>"
	}
}

// exampleKeyFromPattern generates an example key from a regex pattern and parent name
func (g *YAMLGenerator) exampleKeyFromPattern(pattern string, parentKey string) string {
	// For object maps, use {parentKey}0 (e.g., service0, job0)
	if parentKey != "" {
		// Remove final 's' if plural to get singular name
		singularKey := strings.TrimSuffix(parentKey, "s")
		if singularKey != parentKey {
			return singularKey + "0"
		}
		return parentKey + "0"
	}

	// Common patterns in CDS
	switch {
	case strings.Contains(pattern, "^[a-zA-Z0-9"):
		return "my-key"
	case strings.Contains(pattern, "input") || strings.Contains(pattern, "INPUT"):
		return "my-input"
	case strings.Contains(pattern, "output") || strings.Contains(pattern, "OUTPUT"):
		return "my-output"
	case strings.Contains(pattern, "var") || strings.Contains(pattern, "VAR"):
		return "MY_VARIABLE"
	case pattern == ".*":
		return "example-key"
	default:
		// Clean pattern to create a valid key
		cleaned := strings.Trim(pattern, "^$.*[](){}|\\")
		if cleaned != "" && len(cleaned) < 20 {
			return cleaned
		}
		return "key-example"
	}
}

// exampleForFormat returns an example based on JSON Schema format
func (g *YAMLGenerator) exampleForFormat(format string) string {
	examples := map[string]string{
		"date-time": `"2024-01-01T12:00:00Z"`,
		"date":      `"2024-01-01"`,
		"time":      `"12:00:00"`,
		"email":     `"user@example.com"`,
		"hostname":  `"example.com"`,
		"ipv4":      `"192.168.1.1"`,
		"ipv6":      `"::1"`,
		"uri":       `"https://example.com"`,
		"uuid":      `"550e8400-e29b-41d4-a716-446655440000"`,
	}

	if example, ok := examples[format]; ok {
		return example
	}
	return fmt.Sprintf(`"<format: %s>"`, format)
}

// isRequired checks if a property is required
func (g *YAMLGenerator) isRequired(schema *jsonschema.Schema, key string) bool {
	for _, req := range schema.Required {
		if req == key {
			return true
		}
	}
	return false
}

// writeComment writes a multi-line comment if necessary
func (g *YAMLGenerator) writeComment(w io.Writer, comment string, level int) {
	// Limit comment width
	const maxWidth = 80
	indent := g.indentStr(level)
	prefix := indent + g.commentChar + " "

	// Split into lines if too long
	words := strings.Fields(comment)
	if len(words) == 0 {
		return
	}

	var line strings.Builder
	line.WriteString(prefix)

	for _, word := range words {
		if line.Len()+len(word)+1 > maxWidth && line.Len() > len(prefix) {
			fmt.Fprintln(w, line.String())
			line.Reset()
			line.WriteString(prefix)
		}
		if line.Len() > len(prefix) {
			line.WriteString(" ")
		}
		line.WriteString(word)
	}

	if line.Len() > len(prefix) {
		fmt.Fprintln(w, line.String())
	}
}

// indentStr returns the indentation string for a given level
func (g *YAMLGenerator) indentStr(level int) string {
	if level == 0 {
		return ""
	}
	return strings.Repeat(g.indent, level)
}
