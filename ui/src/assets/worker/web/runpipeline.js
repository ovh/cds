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
    loop(2, function () {
        var url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/build/' + buildNumber + '?withArtifacts=true&withTests=true&envName=' + envName;

        var xhr = httpCall(url, api, user, session);
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200 && xhr.responseText !== null) {
            postMessage(xhr.responseText);
            var jsonPb = JSON.parse(xhr.responseText);
            if (jsonPb && jsonPb.status !== 'Building') {
                close();
            }
        }
        return false;
    });
}
