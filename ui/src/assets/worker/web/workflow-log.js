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
    loadLog();
};

function loadLog() {
    loop(2, function () {
        var url = `/project/${key}/workflows/${workflowName}/runs/${number}/nodes/${nodeRunId}/job/${runJobId}/step/${stepOrder}`;
        var xhr = httpCallAPI(url);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200 && xhr.responseText !== null) {
            postMessage(xhr.responseText);
            var jsonLogs = JSON.parse(xhr.responseText);
            if (jsonLogs && jsonLogs.status !== 'Building' && jsonLogs.status !== 'Waiting') {
                close();
            }
        }
        return false;
    });
}
