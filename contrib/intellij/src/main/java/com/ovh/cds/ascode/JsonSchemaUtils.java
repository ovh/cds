package com.ovh.cds.ascode;

import java.util.*;

import com.fasterxml.jackson.databind.JsonNode;
import com.networknt.schema.JsonSchema;
import org.jetbrains.annotations.Nullable;

public class JsonSchemaUtils {

    public static final String JSONSCHEMA_ONEOF = "oneOf";
    public static final String JSONSCHEMA_TYPE = "type";
    public static final String JSONSCHEMA_REF = "$ref";
    public static final String JSONSCHEMA_PROPERTIES = "properties";
    public static final String JSONSCHEMA_OBJECT = "object";
    public static final String JSONSCHEMA_ITEMS = "items";
    public static final String JSONSCHEMA_PATTERN_PROPERTIES = "patternProperties";
    public static final String JSONSCHEMA_PATTERN_ALL = ".*";
    public static final String JSONSCHEMA_REQUIRED = "required";

    static FlatSchema ToFlatSchema(JsonSchema schema) {
        String root = schema.getSchemaNode().findValue(JSONSCHEMA_REF).toString().replace("\"", "");
        List<FlatSchema.FlatElement> flatElts = new ArrayList<>();

        browse(schema, flatElts, root, new ArrayList<>());
        FlatSchema flatSchema = new FlatSchema();
        flatSchema.schema = schema;
        flatSchema.flatElements = flatElts;
        return flatSchema;
    }

    /**
     *  [recursive] Transform JSONSchema to List<FlatSchema.FlatElement> that will be more efficient for completion
     * @param schema Schema to transform
     * @param flatElts Destination structure
     * @param elt Element to transform
     * @param tree Current path of the element
     */
    static private JsonNode browse(JsonSchema schema, List<FlatSchema.FlatElement> flatElts, String elt, List<String> tree) {
        JsonNode currentElt = schema.getRefSchemaNode(elt);
        JsonNode eltProperties = currentElt.get(JSONSCHEMA_PROPERTIES);
        JsonNode eltOneOf = currentElt.get(JSONSCHEMA_ONEOF);
        if (eltProperties == null) {
            return eltOneOf;
        }
        // Browse all item properties
        Iterator<String> ite = eltProperties.fieldNames();
        while (ite.hasNext()) {
            String currentField = ite.next();
            JsonNode prop = eltProperties.get(currentField);
            JsonNode propTypeValue = prop.get(JSONSCHEMA_TYPE);
            String propType = null;
            if (propTypeValue != null) {
                propType = propTypeValue.toString().replace("\"", "");
            }
            JsonNode propPatternProperties = prop.get(JSONSCHEMA_PATTERN_PROPERTIES);
            JsonNode itemsNode = prop.get(JSONSCHEMA_ITEMS);
            JsonNode propRef = prop.get(JSONSCHEMA_REF);
            JsonNode oneOf = prop.get(JSONSCHEMA_ONEOF);

            // Browse PatternProperties/.*/$ref
            if (propType != null && propType.equalsIgnoreCase(JSONSCHEMA_OBJECT) && propPatternProperties != null) {
                // First add current element
                List<String> currentFieldType = new ArrayList<>();
                currentFieldType.add(propType);
                addElement(currentField, flatElts, tree, currentFieldType, null);

                JsonNode patternProperty = propPatternProperties.get(JSONSCHEMA_PATTERN_ALL);

                // Then find Ref def and go into
                if (patternProperty != null) {
                    JsonNode refNode = patternProperty.get(JSONSCHEMA_REF);
                    if (refNode != null) {
                        List<String> newTree = new ArrayList<>(tree);
                        newTree.add(currentField);
                        newTree.add(JSONSCHEMA_PATTERN_ALL);
                        browse(schema, flatElts, refNode.toString().replace("\"", ""), newTree);
                    }
                }
            // typed items without pattern properties
            } else if (propTypeValue != null) {
                JsonNode currentOneOf = null;

                // If there is an path  items/$ref
                if (itemsNode != null) {
                    JsonNode refValue = itemsNode.get(JSONSCHEMA_REF);
                    if (refValue != null) {
                        List<String> newTree = new ArrayList<>(tree);
                        newTree.add(currentField);
                        currentOneOf = browse(schema, flatElts, refValue.toString().replace("\"", ""), newTree);
                    }
                }

                // Then add current element with aggregated oneOf
                List<String> currentFieldType = new ArrayList<>();
                currentFieldType.add(propType);
                addElement(currentField, flatElts, tree, currentFieldType, currentOneOf);

            // untyped element with ref
            } else if (propRef != null) {
                // Going deep into ref and get oneOf deps
                List<String> newTree = new ArrayList<>(tree);
                newTree.add(currentField);
                JsonNode currentOneOf = browse(schema, flatElts, propRef.toString().replace("\"", ""), newTree);

                // Then add current element with aggregated oneOf
                List<String> types = new ArrayList<>();
                types.add(JSONSCHEMA_OBJECT);
                addElement(currentField, flatElts, tree, types, currentOneOf);
            } else {
                List<String> types = new ArrayList<>();
                if (oneOf != null) {
                    Iterator<JsonNode> iteOneOf = oneOf.elements();
                    while(iteOneOf.hasNext()) {
                        JsonNode typeElt = iteOneOf.next().get(JSONSCHEMA_PROPERTIES);
                        if (typeElt != null) {
                            types.add(typeElt.toString().replace("\"", ""));
                        }
                    }
                }
                addElement(currentField, flatElts, tree, types, null);
            }
        }
        return eltOneOf;
    }

    static void addElement(String currentName, List<FlatSchema.FlatElement> flatElements, List<String> tree, List<String> type, @Nullable JsonNode oneOf) {
        FlatSchema.FlatElement flatElement = new FlatSchema.FlatElement();
            flatElement.name = currentName;
            flatElement.depth = tree.size();
            flatElement.types = type;
            flatElement.parents = tree;

            flatElement.oneOf = new HashMap<>();
            if (oneOf != null) {
                Iterator<JsonNode> oneOfIte = oneOf.elements();
                while(oneOfIte.hasNext()) {
                    JsonNode oneOfElt = oneOfIte.next();
                    JsonNode requiredValue =oneOfElt.get(JSONSCHEMA_REQUIRED);
                    if (requiredValue == null) {
                        continue;
                    }
                    List<String> requiredFields = new ArrayList<>();
                    Iterator<String> requiredIte = requiredValue.fieldNames();
                    while(requiredIte.hasNext()) {
                        String requireField = requiredIte.next();
                        requiredFields.add(requireField);
                    }

                    for(String f: requiredFields) {
                        if (!flatElement.oneOf.containsKey(f)) {
                            flatElement.oneOf.put(f, new ArrayList<>());
                        }
                        List<String> current = flatElement.oneOf.get(f);
                        current.addAll(requiredFields);
                        flatElement.oneOf.put(f,current);
                    }
                }
            }
            flatElements.add(flatElement);

    }
}
