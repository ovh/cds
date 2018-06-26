importScripts('../common.js');

var started = false;
var key = '';
var workflowName = '';
var number = 0;
var nodeRunId = 0;
var runJobId = 0;


onmessage = function (e) {
    key = e.data.key;
    workflowName = e.data.workflowName;
    number = e.data.number;
    nodeRunId = e.data.nodeRunId;
    runJobId = e.data.runJobId;

    loadLog(e.data.user, e.data.session, e.data.api);
};

function loadLog (user, session, api) {
    loop(7, function () {
        var url = '/project/' + key + '/workflows/' + workflowName + '/runs/' + number + '/nodes/' + nodeRunId + '/job/' + runJobId + '/log/service';

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
