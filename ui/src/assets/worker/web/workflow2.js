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

        var response = httpCall(url, api, user, session);
        if (response.xhr.status >= 400) {
            return true;
        }
        if (response.xhr.status === 200 && response.xhr.responseText !== null) {
            postMessage(response.xhr.responseText);
        }
        return false;
    });
}
