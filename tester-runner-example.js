// This file deploys a Jenkins instance running quilt-tester. It shows an example
// of all paramters to `tester.New`. It uses a floating IP to automatically
// configure `jenkinsUrl`.

var tester = require("github.com/quilt/tester/tester");

var deployment = createDeployment();
var baseMachine = new Machine({provider: "Amazon"});

deployment.deploy(baseMachine.asMaster());

var worker = baseMachine.asWorker();
worker.floatingIp = "8.8.8.8";
deployment.deploy(worker);

deployment.deploy(tester.New({
    awsAccessKey: "accessKey",
    awsSecretAccessKey: "secret",
    testingNamespace: "quilt-tester",
    slackTeam: "quilt-dev",
    slackChannel: "#testing",
    slackToken: "secret",
    // Because of a bug with the `applyTemplate` function, dollar sign literals (`$`) need to be replaced with `$$`.
    passwordHash: "#jbcrypt:$$2a$$10$$Jcwink6XXoUIp3Ieh.1QR.Mx5idVA7QNLHNcF2jQWhoCA96y5k/jS",
    jenkinsUrl: "http://" + worker.floatingIp + ":8080"
}));
