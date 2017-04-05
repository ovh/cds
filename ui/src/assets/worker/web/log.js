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
    if (user && api) {
        var url = '/project/' + key + '/application/' + appName +
            '/pipeline/' + pipName +'/build/' + buildNumber +
            '/action/' + jobID + '/step/' + stepOrder + '/log';
        if (envName !== '') {
            url += '?envName=' + envName;
        }
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
