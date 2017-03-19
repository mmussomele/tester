var Elasticsearch = require("github.com/quilt/elasticsearch").Elasticsearch;
var infrastructure = require("github.com/quilt/tester/config/infrastructure")

var deployment = createDeployment({});
deployment.deploy(infrastructure);
deployment.deploy(new Elasticsearch(infrastructure.nWorker).public());
