(function(mod) {
    if (typeof exports == "object" && typeof module == "object") // CommonJS
        mod(require("../../lib/codemirror"));
    else if (typeof define == "function" && define.amd) // AMD
        define(["../../lib/codemirror"], mod);
    else // Plain browser env
        mod(CodeMirror);
})(function(CodeMirror) {
    "use strict";
    CodeMirror.registerHelper("lint", "yaml", function(text) {
        let errors = [];
        try { jsyaml.loadAll(text); }
        catch(e) {
            let loc = e.mark,
                from = loc ? CodeMirror.Pos(loc.line, loc.column) : CodeMirror.Pos(0, 0),
                to = from;
            errors.push({ from: from, to: to, message: e.message });
        }
        return errors;
    });

});
