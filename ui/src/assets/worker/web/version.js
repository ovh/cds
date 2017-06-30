importScripts('../common.js');

onmessage = function () {
    loadVersion();
};

function loadVersion () {
    loop(10, function () {
        var response = httpCall('assets/version.json?ts=' + (new Date()).getTime(), '../../../');
        if (response.xhr.status >= 400) {
            return true;
        }
        if (response.xhr.status === 200) {
            postMessage(response.xhr.responseText);
        }
        return false;
    });
}
