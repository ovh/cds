importScripts('../common.js');

var key = '';
var workflowName = '';
var number = '';

var previousLastModified;
var previousId;

onmessage = function (e) {
    key = e.data.key;
    workflowName = e.data.workflowName;
    number = e.data.number;
    loadWorkflow(e.data.user, e.data.session, e.data.api);
};

function loadWorkflow (user, session, api) {
    loop(5, function () {
        var url = '/project/' + key + '/workflows/' + workflowName + '/runs/' + number;

        var xhr = httpCall(url, api, user, session);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200 && xhr.responseText !== null) {
            var wr = JSON.parse(xhr.responseText);
            if (previousLastModified && wr.last_modified === previousLastModified && previousId === wr.id) {
                return;
            }
            previousLastModified = wr.last_modified;
            previousId = wr.id
            postMessage(xhr.responseText);
        }
        return false;
    });
}
