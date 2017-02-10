var apiTestEnv = 'foo.bar'; // url for test cf environments/environment.ts

function httpCall (path, host, user) {
    if (host !== apiTestEnv) {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', host + path, false, null, null);
        xhr.setRequestHeader("Authorization", "Basic " + user.token);
        xhr.send(null);
        if (xhr.status === 200) {
            return xhr.responseText;
        }
        return null;
    }
}

function httpCallSharedWorker (path, host, user, caller, k, callback) {
    if (host !== apiTestEnv) {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', host + path, false, null, null);
        xhr.setRequestHeader("Authorization", "Basic " + user.token);

        xhr.onload = function (e) {
            if (xhr.readyState === 4) {
                if (xhr.status === 200) {
                    // for each subscription, give response
                    caller.ports.forEach(function (p) {
                        ports[p-1].postMessage(xhr.responseText);
                    });
                    callback(k, xhr.responseText);
                }
            }
        };
        xhr.onerror = function (e) {
            console.error(xhr.statusText);
        };
        xhr.send(null);
    }
}
