importScripts('../common.js');

var key = '';
var workflowName = '';
var number = 0;
var limit = 10;

onmessage = function (e) {
    key = e.data.key;
    workflowName = e.data.workflowName;
    number = e.data.number;
    limit = e.data.limit;
    loadWorkflowRuns(e.data.user, e.data.session, e.data.api);
};

function loadWorkflowRuns(user, session, api) {
    loop(10, function () {
        var url = '/project/' + key + '/workflows/' + workflowName + '/runs';

        if (limit != null) {
            url += '?limit=' + limit;
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
