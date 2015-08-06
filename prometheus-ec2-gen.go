package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/ec2"
)

type TargetGroup struct {
	Labels  map[string]string
	Targets []string
}

func main() {
	tags := []string{"Type", "Deployment", "Version"}

	auth, err := aws.EnvAuth()
	if err != nil {
		log.Fatal(err)
	}

	filter := ec2.NewFilter()
	for _, t := range tags {
		filter.Add("tag-key", t)
	}
	e := ec2.New(auth, aws.USWest2)
	resp, err := e.Instances(nil, filter)
	instances := flattenReservations(resp.Reservations)

	targetGroups := make(map[string]*TargetGroup)

	for _, instance := range instances {
		if instance.State.Code != 16 { // 16 = Running
			continue
		}

		key := ""
		for _, tagKey := range tags {
			key = fmt.Sprintf("%s|%s=%s", key, tagKey, getTag(instance, tagKey))
		}

		targetGroup, ok := targetGroups[key]
		if !ok {
			labels := make(map[string]string)
			for _, tagKey := range tags {
				tagVal := getTag(instance, tagKey)
				if tagVal != "" {
					labels[tagKey] = tagVal
				}
			}
			targetGroup = &TargetGroup{
				Labels:  labels,
				Targets: make([]string, 0),
			}
			targetGroups[key] = targetGroup
		}

		target := fmt.Sprintf("%s:3000", instance.PrivateIpAddress)
		targetGroup.Targets = append(targetGroup.Targets, target)
	}

	const conf = `
global:
  scrape_interval:     15s
  evaluation_interval: 15s

scrape_configs:
  # Prometheus itself
  - job_name: 'prometheus'

    target_groups:
      - targets: ['localhost:9090']

  # src
  - job_name:       'src'
    scrape_interval: 5s
    target_groups:
{{ range .TargetGroups }}
      - targets: {{ marshal .Targets }}
        labels:
{{ range $labelKey, $labelValue := .Labels }}
          {{ $labelKey }}: {{ $labelValue }}{{ end }}
{{ end }}
`
	templateVars := struct {
		TargetGroups map[string]*TargetGroup
	}{
		TargetGroups: targetGroups,
	}

	funcMap := template.FuncMap{
		"marshal": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
	}

	t := template.Must(template.New("prometheus.yml").Funcs(funcMap).Parse(conf))
	t.Execute(os.Stdout, templateVars)
}

func getTag(instance ec2.Instance, key string) string {
	for _, t := range instance.Tags {
		if t.Key == key {
			return t.Value
		}
	}
	return ""
}

func flattenReservations(reservations []ec2.Reservation) []ec2.Instance {
	instances := make([]ec2.Instance, 0)
	for _, r := range reservations {
		instances = append(instances, r.Instances...)
	}
	return instances
}
