var etcd = require("github.com/quilt/etcd");
var infrastructure = require("github.com/quilt/tester/config/infrastructure")

var deployment = createDeployment({});
deployment.deploy(infrastructure);
deployment.deploy(new etcd.Etcd(infrastructure.nWorker*2));
