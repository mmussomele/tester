var nginx = require("github.com/quilt/nginx");
var infrastructure = require("github.com/quilt/tester/config/infrastructure")

var deployment = createDeployment({});
deployment.deploy(infrastructure);

for (var i = 0 ; i < infrastructure.nWorker ; i++) {
    deployment.deploy(nginx.New(80));
    deployment.deploy(nginx.New(8000));
}
