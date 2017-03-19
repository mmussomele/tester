var spark = require("github.com/quilt/spark");
var infrastructure = require("github.com/quilt/tester/config/infrastructure");

var deployment = createDeployment({});
deployment.deploy(infrastructure);

var sprk = new spark.Spark(1, 3);

deployment.deploy(sprk);
