importScripts('../common.js');

var started = false;
var key = '';
var appName = '';
var pipName = '';
var buildNumber = '';
var jobID = 0;
var stepOrder = 0;

onmessage = function (e) {
    key = e.data.key;
    appName = e.data.appName;
    pipName = e.data.pipName;
    buildNumber = e.data.buildNumber;
    jobID = e.data.jobID;
    stepOrder = e.data.stepOrder;
    loadLog(e.data.user, e.data.api);
};

function loadLog (user, api) {
    if (user && api) {
        setInterval(function () {
            var url = '/project/' + key + '/application/' + appName +
                '/pipeline/' + pipName +'/build/' + buildNumber +
                '/action/' + jobID + '/step/' + stepOrder + '/log';
            var stepLogs = httpCall(url, api, user);
            postMessage(stepLogs);

            var jsonLogs = JSON.parse(stepLogs);
            if (jsonLogs.status !== 'Building') {
                close();
            }
        }, 2000);
    }
}
