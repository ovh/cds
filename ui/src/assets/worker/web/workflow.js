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
    var url = '/project/' + key + '/application/' + appName +
        '?applicationStatus=true&branchName=' + branch + '&version=' + version;

    if (user && api) {
        postMessage(httpCall(url, api, user));
        setInterval(function () {
            var response = httpCall(url, api, user);
            postMessage(response);
        }, 2000);
    }
}
