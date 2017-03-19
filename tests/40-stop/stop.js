// Place a google/pause container on each worker machine.

var infrastructure = require("github.com/quilt/tester/config/infrastructure")

var deployment = createDeployment({});
deployment.deploy(infrastructure);

var containers = new Service("containers",
    new Container("google/pause").replicate(infrastructure.nWorker));
containers.place(new LabelRule(true, containers));

deployment.deploy(containers);
