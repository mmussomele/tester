// XXX: Docker does not currently support uploading files to containers with a
// UID other than root. To get around this, we upload our files to a staging
// directory, and copy them into the Jenkins home directory (as the Jenkins
// user) before starting Jenkins.
var jenkinsStagingDir = "/tmp/files/"

exports.New = function(opts) {
    assertRequiredParameters(opts, ["awsAccessKey", "awsSecretAccessKey", "testingNamespace", "slackTeam", "slackChannel", "slackToken"]);

    var container = new Container("quilt/tester",
        ["/bin/bash", "-c",
            "cp -r " + jenkinsStagingDir + "* /var/jenkins_home;" +
            "/bin/tini -s -- /usr/local/bin/jenkins.sh"]);
    container.setEnv("AWS_ACCESS_KEY", opts.awsAccessKey);
    container.setEnv("AWS_SECRET_ACCESS_KEY", opts.awsSecretAccessKey);
    container.setEnv("TESTING_NAMESPACE", opts.testingNamespace);
    container.setEnv("TZ", "/usr/share/zoneinfo/America/Los_Angeles");
    setupFiles(container, opts);

    var jenkins = new Service("jenkins", [container]);

    // Allow inbound connections to the Jenkins web UI.
    publicInternet.connect(8080, jenkins);

    // The tests talk to the deployed machines on various ports. We allow them here.
    jenkins.connect(22, publicInternet); // Required by `quilt ssh`.
    jenkins.connect(80, publicInternet); // Required by network tests.
    jenkins.connect(443, publicInternet); // Required by network tests.
    jenkins.connect(8000, publicInternet); // Required by network tests.
    jenkins.connect(9200, publicInternet); // Required by the elasticsearch test.
    jenkins.connect(9000, publicInternet); // Required by the Quilt daemon for API communication.
    jenkins.connect(9999, publicInternet); // Required by the Quilt daemon for minion communcation.

    return jenkins;
}

function setupFiles(jenkins, opts) {
    var files = [];

    var rootConfig = new File("config.xml", read("config/jenkins/root.xml"));
    files.push(rootConfig);

    var goConfig = new File("org.jenkinsci.plugins.golang.GolangBuildWrapper.xml",
        read("config/jenkins/go.xml"));
    files.push(goConfig);

    var jobConfig = new File("jobs/quilt-tester/config.xml",
        applyTemplate(read("config/jenkins/job.xml"),
            {slackTeam: opts.slackTeam,
             slackToken: opts.slackToken,
             slackChannel: opts.slackChannel}));
    files.push(jobConfig);

    if (opts.passwordHash !== undefined) {
        var adminConfig = new File("users/admin/config.xml",
            applyTemplate(read("config/jenkins/admin.xml"),
                {passwordHash: opts.passwordHash}));
        files.push(adminConfig);
    }

    if (opts.jenkinsUrl !== undefined) {
        var locConfig = new File("jenkins.model.JenkinsLocationConfiguration.xml",
            applyTemplate(read("config/jenkins/location.xml"),
                {jenkinsUrl: opts.jenkinsUrl}));
        files.push(locConfig);
    }

    files.forEach(function(f) {
        jenkins.filepathToContent[jenkinsStagingDir + f.path] = f.content
    });
}

// applyTemplate replaces the keys defined by `vars` with their corresponding
// values in `template`. A variable is denoted in the template using {{key}}.
function applyTemplate(template, vars) {
    for (k in vars) {
        template = template.replace("{{" + k + "}}", vars[k]);
    }
    return template;
}

function assertRequiredParameters(opts, requiredKeys) {
    requiredKeys.forEach(function(key) {
        if (opts[key] === undefined) {
            throw key + " is required"
        }
    })
}

function File(path, content) {
    this.path = path;
    this.content = content;
}
