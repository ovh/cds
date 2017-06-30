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

        var response = httpCall(url, api, user, session);
        if (response.xhr.status >= 400) {
            return true;
        }
        if (response.xhr.status === 200 && response.xhr.responseText !== null) {
            postMessage(response.xhr.responseText);
            var jsonPb = JSON.parse(response.xhr.responseText);
            if (jsonPb && jsonPb.status !== 'Building') {
                close();
            }
        }
        return false;
    });
}
