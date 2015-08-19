Prometheus EC2 Service Discovery
================================

`prometheus-ec2-discovery` generates target groups by querying the EC2 API and
grouping your instances based on their tags. This is to be used in conjunction
with Prometheus's [custom service discovery][1] feature.

Example
-------

```
$ prometheus-ec2-discovery --port 9100 --tags=Type,Deployment
[
  {
    "targets": ["172.22.2.57:9100"],
    "labels": {"Deployment": "staging", "Type": "frontend"}
  },
  {
    "targets": ["172.22.1.81:9100", "172.22.1.149:9100", "172.22.2.142:9100"],
    "labels": {"Deployment": "production", "Type": "frontend"}
  },
  {
    "targets": ["172.22.1.121:9100"],
    "labels": {"Deployment": "staging", "Type": "backend"}
  },
  {
    "targets": ["172.22.2.245:9100", "172.22.2.248:9100"],
    "labels": {"Deployment": "production", "Type": "backend"}
  }
]
```

This generates to stdout all ec2 instances with either the `Type` or
`Deployment` tag, and then labeled via the tag value. Note that the target ips
are the `PrivateIpAddress` and that you must ensure prometheus is in the
correct security groups to access the instances.

Install
-------

First install the binary

```
$ go get github.com/keegancsmith/prometheus-ec2-discovery
```

Then configure a [file based service discovery] [1] target group in your
`prometheus.yml` config:

```yaml
scrape_configs:
  - job_name: my_ec2_service

    file_sd_configs:
    - names: ['tgroups/*.json']
```

You can then setup a crontab job to update the target groups every minute:

```
* * * * * cd /path/to/prometheus/dir && . ./aws.inc && prometheus-ec2-discovery --tags=Name --dest=tgroups/ec2.json
```

In the example crontab command, aws.inc contains the AWS_* access keys environment
variables.

All Tags
--------

If you specify empty `--tags=`, then all tags are used.


Docker
------

```
$ docker run --rm -e AWS_SECRET_ACCESS_KEY -e AWS_ACCESS_KEY keegancsmith/prometheus-ec2-discovery
```

[1]: http://prometheus.io/blog/2015/06/01/advanced-service-discovery/#custom-service-discovery "Custom Service Discovery"
