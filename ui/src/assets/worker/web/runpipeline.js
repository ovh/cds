importScripts('../common.js');

var started = false;
var pipName = '';
var key = '';
var appName = '';
var buildNumber = 0;
var envName = '';

onmessage = function (e) {
    buildNumber = e.data.buildNumber;
    key = e.data.key;
    appName = e.data.appName;
    pipName = e.data.pipName;
    envName = e.data.envName;
    loadBuild(e.data.user, e.data.session, e.data.api);
};

function loadBuild (user, session, api) {
    if (user && api) {
        var url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/build/' + buildNumber + '?withArtifacts=true&withTests=true&envName=' + envName;
        postMessage(httpCall(url, api, user, session));

        setInterval(function () {
            var response = httpCall(url, api, user, session);
            postMessage(response);
            var jsonPb = JSON.parse(response);
            if (jsonPb.status !== 'Building') {
                close();
            }
        }, 2000);
    }
}
