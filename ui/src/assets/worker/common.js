var apiTestEnv = 'foo.bar'; // url for test cf environments/environment.ts

function httpCall (path, host, user, session) {
    if (host !== apiTestEnv) {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', host + path, false, null, null);
        if (session) {
            xhr.setRequestHeader("Session-Token", session);
        } else {
            xhr.setRequestHeader("Authorization", "Basic " + user.token);
        }

        xhr.send(null);
        if (xhr.status === 200) {
            return xhr.responseText;
        }
        return null;
    }
}
