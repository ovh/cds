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
    loadWorkflow(e.data.user, e.data.session, e.data.api);
};

function loadWorkflow (user, session, api) {
    var url = '/project/' + key + '/application/' + appName +
        '?applicationStatus=true&branchName=' + branch + '&version=' + version;

    if (user && api) {
        postMessage(httpCall(url, api, user, session));
        setInterval(function () {
            var response = httpCall(url, api, user, session);
            postMessage(response);
        }, 2000);
    }
}
