importScripts('../common.js');

var started = false;
var key = '';
var appName = '';
var pipName = '';
var buildNumber = '';
var jobID = 0;
var stepOrder = 0;
var envName = '';

onmessage = function (e) {
    key = e.data.key;
    appName = e.data.appName;
    pipName = e.data.pipName;
    buildNumber = e.data.buildNumber;
    envName = e.data.envName;
    jobID = e.data.jobID;
    stepOrder = e.data.stepOrder;
    loadLog(e.data.user, e.data.session, e.data.api);
};

function loadLog (user, session, api) {
    loop(2, function () {
        var url = '/project/' + key + '/application/' + appName +
            '/pipeline/' + pipName + '/build/' + buildNumber +
            '/action/' + jobID + '/step/' + stepOrder + '/log';
        if (envName !== '') {
            url += '?envName=' + envName;
        }

        var xhr = httpCall(url, api, user, session);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200 && xhr.responseText) {
            postMessage(xhr.responseText);
            var jsonLogs = JSON.parse(xhr.responseText);
            if (jsonLogs && jsonLogs.status !== 'Building') {
                close();
            }
        }
        return false;
    });
}
