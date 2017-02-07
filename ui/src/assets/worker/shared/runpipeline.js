importScripts('../common.js');

var workerStarted = false;
var ports = [];
var call = {};

onconnect = function (e) {
    if (!e.ports || e.ports.length === 0) {
        return;
    }
    // Register worker + give it an ID
    ports.push(e.ports[0]);
    var id = ports.length;

    // Return ID to caller
    e.ports[0].postMessage({ 'worker_id': id });

    // Receive msg
    e.ports[0].onmessage = function (e) {
        switch (e.data.action) {
            case 'subscribe':
                addCall(e);
                break;
            case 'unsubscribe':
                removePort(e);
                break;
        }
        // Run worker
        if (!workerStarted) {
            workerStarted = true;
            loadBuild(e.data.user, e.data.api);
        }
    };
};

/**
 * Load build
 * @param user
 * @param api
 */
function loadBuild (user, api) {
    if (user && api) {
        callAPI(user, api);
        setInterval(function () {
            callAPI(user, api);
        }, 2000);
    }
}

function callAPI(user, api) {
    // Browse all call needed
    for(var k in call){
        var c = call[k];
        var url = '/project/' + c.key+ '/application/' + c.appName +'/pipeline/' + c.pipName + '/build/' + c.buildNumber + '?withArtifacts=true&withTests=true&envName=' + c.envName;
        httpCallSharedWorker(url, api, user, c, k, postCall);
    }
}

function postCall(k, response) {
    var jsonLogs = JSON.parse(response);
    if (jsonLogs.status !== 'Building') {
        delete call[k];
    }
}

/**
 * Add a port for a call
 * @param e
 */
function addCall(e) {
    // call ID
    var key = 'filter-' + e.data.key + '-' + e.data.appName + '-' + e.data.pipName + '-' + e.data.buildNumber + '-' + e.data.envName;

    // If call don't exist, create it
    if (!call[key]) {
        call[key]= {
            key: e.data.key,
            appName: e.data.appName,
            pipName: e.data.pipName,
            buildNumber: e.data.buildNumber,
            envName: e.data.envName,
            ports: []
        };
    }

    // Add port for the call
    call[key].ports.push(e.data.id);
}

/**
 * Remove a port from all call
 * @param e
 */
function removePort(e) {
    // Browse all call
    for (var k in call) {
        var c = call[k];
        // For each call, browse all ports
        for (var indexPort = 0; indexPort < c.ports.length; indexPort++) {
            // If id match, delete it
            if (c.ports[indexPort] === e.data.id) {
                c.ports.splice(indexPort, 1);
                indexPort--;
            }
        }
        if (c.ports.length === 0) {
            delete call[k];
        }
    }
}


