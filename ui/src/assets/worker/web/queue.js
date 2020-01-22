importScripts('../common.js');

var started = false;
var status = [];

onmessage = function (e) {
    status = e.data.status;
    if (!started) {
        loadWorkflowRuns();
    }
};

function loadWorkflowRuns() {
    started = true;
    loop(5, function () {
        var url = '/queue/workflows';
        if (status && status.length > 0) {
            url = url.concat('?', status.map(function (s) { return 'status=' + s; }).join('&'))
        }
        var xhr = httpCallAPI(url);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status <= 300 && xhr.responseText !== null) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
