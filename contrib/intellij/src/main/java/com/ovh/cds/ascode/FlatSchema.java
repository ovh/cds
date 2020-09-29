package com.ovh.cds.ascode;

import com.networknt.schema.JsonSchema;

import java.util.List;
import java.util.Map;

public class FlatSchema {
    public JsonSchema schema;
    public List<FlatElement> flatElements;

    public static class FlatElement {
        public int depth;
        public String name;
        public List<String> types;
        public List<String> parents;
        public Map<String, List<String>> oneOf;
    }
}


