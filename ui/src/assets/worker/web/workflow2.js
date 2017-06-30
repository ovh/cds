importScripts('../common.js');

var key = '';
var workflowName = '';
var number = '';

onmessage = function (e) {
    key = e.data.key;
    workflowName = e.data.workflowName;
    number = e.data.number;
    loadWorkflow(e.data.user, e.data.session, e.data.api);
};

function loadWorkflow (user, session, api) {
    loop(2, function () {
        var url = '/project/' + key + '/workflows/' + workflowName + '/runs/' + number;

        var xhr = httpCall(url, api, user, session);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200 && xhr.responseText !== null) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
