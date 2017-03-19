var infrastructure = require("github.com/quilt/tester/config/infrastructure")

var deployment = createDeployment();
deployment.deploy(infrastructure);

var connected = new Service("connected",
    new Container("alpine", ["tail", "-f", "/dev/null"])
        .replicate(infrastructure.nWorker*2)
);
connected.connect(80, publicInternet);

var notConnected = new Service("not-connected",
    new Container("alpine", ["tail", "-f", "/dev/null"])
        .replicate(infrastructure.nWorker*2)
);

deployment.deploy([connected, notConnected]);
