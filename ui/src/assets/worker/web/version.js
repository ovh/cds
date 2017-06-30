importScripts('../common.js');

onmessage = function () {
    loadVersion();
};

function loadVersion () {
    loop(10, function () {
        var xhr = httpCall('assets/version.json?ts=' + (new Date()).getTime(), '../../../');
        if (xhr.status >= 400) {
            return true;
        }
        if (xhr.status === 200) {
            postMessage(xhr.responseText);
        }
        return false;
    });
}
