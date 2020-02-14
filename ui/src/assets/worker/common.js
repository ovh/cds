function loop(initWaitingTime, func) {
    var timeToWait = initWaitingTime * 1000;

    // First call  then run in an interval
    func();
    setInterval(function () {
        func();
    }, timeToWait);
}

function httpCallAPI(path) {
    return httpCall('./../../../cdsapi', path)
}

function httpCall(host, path) {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', host + path, false, null, null);
    xhr.send(null);
    if (xhr.status === 401 || xhr.status === 403 || xhr.status === 404) {
        close();
    }
    return xhr;
}

function fibonacci(retry) {
    if (retry <= 1) {
        return retry;
    }
    return fibonacci(retry - 1) + fibonacci(retry - 2);
}

function connectSSE(url, headAuthKey, headAuthValue) {
    var headers = {};
    headers[headAuthKey] = headAuthValue;
    return new EventSourcePolyfill(url, { headers: headers, errorOnTimeout: false, checkActivity: false, heartbeatTimeout: 300000 });
}