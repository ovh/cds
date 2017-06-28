importScripts('../common.js');

var started = false;
var key = '';
var workflowName = '';
var number = 0;
var nodeRunId = 0;
var runJobId = 0;
var stepOrder = 0;


onmessage = function (e) {
    key = e.data.key;
    workflowName = e.data.workflowName;
    number = e.data.number;
    nodeRunId = e.data.nodeRunId;
    runJobId = e.data.runJobId;
    stepOrder = e.data.stepOrder;

    loadLog(e.data.user, e.data.session, e.data.api);
};

function loadLog (user, session, api) {
    if (user && api) {
        var url = '/project/' + key + '/workflows/' + workflowName + '/runs/' + number+ '/nodes/' + nodeRunId + '/job/' + runJobId + '/step/' + stepOrder;
        postMessage(httpCall(url, api, user, session));
        setInterval(function () {
            var stepLogs = httpCall(url, api, user, session);
            postMessage(stepLogs);
            var jsonLogs = JSON.parse(stepLogs);
            if (jsonLogs && jsonLogs.status !== 'Building') {
                close();
            }
        }, 2000);
    }
}
