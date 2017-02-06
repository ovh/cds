importScripts('../common.js');

var started = false;
var branch = '';
var key = '';
var appName = '';
var version = 0;

onmessage = function (e) {
    branch = e.data.branch;
    key = e.data.key;
    appName = e.data.appName;
    version = e.data.version;
    loadWorkflow(e.data.user, e.data.api);
};

function loadWorkflow (user, api) {
    if (user && api) {
        setInterval(function () {
            var url = '/project/' + projectKey + '/application/' + applicationName +
                '?applicationStatus=true&branchName=' + branchName + '&version=' + version;
            postMessage(httpCall(url, api, user));
        }, 2000);
    }
}
