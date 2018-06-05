importScripts('../common.js');

onmessage = function (e) {
    loadWorkflowRuns(e.data.user, e.data.session, e.data.api);
};

function loadWorkflowRuns(user, session, api) {
    loop(5, function () {
        var url = '/queue/workflows';

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
