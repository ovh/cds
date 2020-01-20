importScripts('../common.js');

var started = false;
var branch = '';
var key = '';
var appName = '';
var version = 0;
var remote = '';

onmessage = function (e) {
    branch = e.data.branch;
    key = e.data.key;
    appName = e.data.appName;
    version = e.data.version;
    remote = e.data.remote;
    loadWorkflow();
};

function loadWorkflow() {
    loop(2, function () {
        var url = `/project/${key}/application/${appName}/tree/status?branchName=${encodeURIComponent(branch)}&remote=${encodeURIComponent(remote)}&version=${version}`;
        var xhr = httpCallAPI(url);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200 && xhr.responseText !== null) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
