importScripts('../common.js');

var started = false;
var status = [];

onmessage = function (e) {
    status = e.data.status;
    if (!started) { loadWorkflowRuns(e.data.user, e.data.session, e.data.api); }
};

function loadWorkflowRuns(user, session, api) {
    started = true;
    loop(5, function () {
        var url = '/queue/workflows';

        if (status && status.length > 0) {
            url = url.concat('?', status.map(function (s) { return "status=" + s; }).join('&'))
        }

        var xhr = httpCall(url, api, user, session);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status <= 300 && xhr.responseText !== null) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}