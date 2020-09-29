package com.ovh.cds.ascode;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import com.intellij.codeInsight.completion.*;
import com.intellij.codeInsight.lookup.LookupElementBuilder;
import com.intellij.openapi.editor.Document;
import com.intellij.openapi.editor.Editor;
import com.intellij.openapi.editor.VisualPosition;
import com.intellij.patterns.PlatformPatterns;
import com.intellij.util.ProcessingContext;
import com.networknt.schema.JsonSchema;
import com.networknt.schema.JsonSchemaFactory;
import com.networknt.schema.SpecVersion;

import java.util.*;

import org.codehaus.plexus.util.StringUtils;
import org.jetbrains.annotations.NotNull;
import org.jetbrains.yaml.YAMLLanguage;

import java.io.*;
import java.util.stream.Collectors;

public class AsCodeCompletion extends CompletionContributor {

    public AsCodeCompletion() {
        extend(CompletionType.BASIC, PlatformPatterns.psiElement().withLanguage(YAMLLanguage.INSTANCE), new Provider());
    }

    private static class Provider extends CompletionProvider<CompletionParameters> {
        FlatSchema fs;
        public Provider() {
            ObjectMapper jsonObjectMapper = new ObjectMapper(new YAMLFactory());
            JsonSchemaFactory factory = JsonSchemaFactory.builder(JsonSchemaFactory.getInstance(SpecVersion.VersionFlag.V4)).objectMapper(jsonObjectMapper).build();
            try {
                String home = System.getProperty("user.home");
                InputStream jsonFile = new FileInputStream(home + "/.cds-schema/workflow.schema.json");
                JsonSchema schema = factory.getSchema(jsonFile);
                this.fs = JsonSchemaUtils.ToFlatSchema(schema);
            } catch (FileNotFoundException e) {
                e.printStackTrace();
            }
        }

        @Override
        protected void addCompletions(@NotNull CompletionParameters parameters, @NotNull ProcessingContext context, @NotNull CompletionResultSet result) {
            final Editor editor = parameters.getEditor();
            final Document doc = editor.getDocument();

            VisualPosition caretPosition = editor.getCaretModel().getPrimaryCaret().getVisualPosition();
            String[] lines = doc.getText().split("\n");
            if (lines.length < caretPosition.line + 1) {
                return;
            }
            List<String> autoCompleteResponse = new ArrayList<>();
            String currentLine = lines[caretPosition.line];
            int firstColon = currentLine.indexOf(':');
            if (firstColon == -1){
                autoCompleteResponse = autoCompleteKey(currentLine, caretPosition, lines);
            }
            for(String v: autoCompleteResponse) {
                LookupElementBuilder elt = LookupElementBuilder.create(v);
                result.addElement(elt);
            }
        }


        private List<String> autoCompleteKey(String currentLine, VisualPosition pos, String[] lines) {
            // Find yaml level
            int depth = findDepth(currentLine);
            if (depth == -1) {
                return new ArrayList<>();
            }

            return findKeySuggestion(currentLine, depth, pos, lines);
        }

        private int findDepth(String text) {
            int spaceNumber = 0;
            for(int i=0; i<text.length(); i++) {
                char ch = text.charAt(i);
                if (ch == ' ' || ch == '-') {
                    spaceNumber++;
                } else {
                    break;
                }
            }

            int depth = -1;
            if (spaceNumber%2 == 0) {
                depth = spaceNumber/2;
            }
            return depth;
        }

        private List<String> findParent(int currentLine, int depth, String[] lines) {
            List<String> parents = new ArrayList<>();
            int refDepth = depth;

            for (int i = currentLine; i > 0; i--) {
                String currentText = lines[i];
                if (currentText.indexOf(':') == -1) {
                    continue;
                }

                // if has key, find indentation
                int currentLintDepth = findDepth(currentText);
                if (currentLintDepth >= refDepth) {
                    continue;
                }
                // find parent key
                String pkey = currentText.substring(0, currentText.indexOf(':')).trim();
                parents.add(0, pkey);
                refDepth = currentLintDepth;
                if (refDepth == 0) {
                    break;
                }
            }
            return parents;
        }

        private List<String> findNeighbour(String currentLine, VisualPosition pos, String[] lines) {
            List<String> neighbours = new ArrayList<>();
            int nbOfSpaces = currentLine.length() - StringUtils.stripStart(currentLine, " ").length();

            if (pos.line > 0) {
                // find neighbour before
                for (int i = pos.line -1; i >= 0; i--) {
                    String line = lines[i];
                    String currentText = StringUtils.stripStart(line, " ");
                    int currentSpace = line.length() - currentText.length();
                    if (currentSpace != nbOfSpaces) {
                        // check if we are in a array
                        if (currentSpace + 2 != nbOfSpaces || currentText.indexOf('-') != 0) {
                            break;
                        }
                        currentText = StringUtils.stripStart(currentText.substring(1), "");
                    } else if (currentText.indexOf('-') == 0) {
                        continue;
                    }
                    neighbours.add(currentText.split(":")[0]);
                }
            }
            if (pos.line < lines.length - 1) {
                // find neighbour before
                for (int i = pos.line + 1; i < lines.length; i++) {
                    String line = lines[i];
                    String currentText = StringUtils.stripStart(line, " ");
                    int currentSpace = line.length() - currentText.length();
                    if (currentSpace != nbOfSpaces) {
                        break;
                    }
                    neighbours.add(currentText.split(":")[0]);
                }
            }
            return neighbours;
        }

        private List<String> findKeyToExclude(String currentLine, VisualPosition pos, String[] lines, String lastParent) {
            // Find neighbour to know which keys are already here
            List<String> neighbours = findNeighbour(currentLine, pos, lines);

            // Exclude key from oneOf.required
            List<String> keyToExclude = new ArrayList<>();

            FlatSchema.FlatElement parent = this.fs.flatElements.stream().filter(e -> e.name.equals(lastParent)).findFirst().orElse(null);
            if (parent != null && parent.oneOf != null && parent.oneOf.size() > 0) {
                for (String key : neighbours) {
                    if (!parent.oneOf.containsKey(key)) {
                        continue;
                    }
                    for (Map.Entry<String, List<String>> es : parent.oneOf.entrySet()) {
                        keyToExclude = es.getValue().stream().filter(k -> parent.oneOf.get(key).stream().noneMatch(kk -> kk.equals(k))).collect(Collectors.toList());
                    }
                }
            }

            // Add neighbour in exclude array
            keyToExclude.addAll(neighbours);
            return keyToExclude;
        }

        private List<String> findKeySuggestion(String currentLine, int depth, VisualPosition pos, String[] lines) {
            // Get all keys that match the depth
            List<FlatSchema.FlatElement> eltMatchesLevel = this.fs.flatElements.stream()
                    .filter(elt -> elt.depth == depth)
                    .collect(Collectors.toList());

            // Filter by parent
            List<String> parents = findParent(pos.line, depth, lines);
            if (depth > 0) {
                // Find element that match the same parents
                eltMatchesLevel = eltMatchesLevel.stream()
                        .filter(elt -> {
                            if (elt.parents.size() != parents.size()) {
                                return false;
                            }
                            for (int i = 0; i<elt.parents.size(); i++) {
                                String currentParent = elt.parents.get(i);
                                if (currentParent.equals(".*")) {
                                    continue;
                                }
                                if (!currentParent.equals(parents.get(i))) {
                                    return false;
                                }
                            }
                            return true;
                        })
                        .collect(Collectors.toList());
            }


            // Find key to exclude from suggestion
            String lastParent = "";
            if (parents.size() > 0) {
                lastParent = parents.get(parents.size() -1);
            }
            List<String> keyToExclude = findKeyToExclude(currentLine, pos, lines, lastParent);
            eltMatchesLevel = eltMatchesLevel.stream().filter(elt -> !keyToExclude.contains(elt.name)).collect(Collectors.toList());
            return eltMatchesLevel.stream().map(e -> e.name + ": ").collect(Collectors.toList());
        }
    }
}
