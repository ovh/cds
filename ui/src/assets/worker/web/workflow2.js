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
    var url = '/project/' + key + '/workflows/' + workflowName + '/runs/' + number;
    if (user && api) {
        postMessage(httpCall(url, api, user, session));
        setInterval(function () {
            var response = httpCall(url, api, user, session);
            postMessage(response);
        }, 2000);
    }
}
