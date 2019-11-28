(function(mod) {
    if (typeof exports == "object" && typeof module == "object") // CommonJS
        mod(require("../../lib/codemirror"));
    else if (typeof define == "function" && define.amd) // AMD
        define(["../../lib/codemirror"], mod);
    else // Plain browser env
        mod(CodeMirror);
})(function(CodeMirror) {
    "use strict";
    CodeMirror.registerHelper("lint", "workflow-schema", function(text, options) {
        let v = new Validator();
        const yamlData = yaml.load(text);
        let result = v.validate(yamlData, options.schema);
        return result.errors;
    });
});
