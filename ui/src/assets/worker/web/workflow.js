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
    loop(2, function () {
        var url = '/project/' + key + '/application/' + appName +
            '?withSchedulers=true&withPollers=true&applicationStatus=true&branchName=' + branch + '&version=' + version;

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
