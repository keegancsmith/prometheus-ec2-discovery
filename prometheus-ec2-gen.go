package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/ec2"
)

var (
	tags   []string
	region aws.Region
	port   int
)

// TargetGroup is a collection of related hosts that prometheus monitors
type TargetGroup struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

func main() {
	initFlags()

	filter := ec2.NewFilter()
	for _, t := range tags {
		filter.Add("tag-key", t)
	}

	auth, err := aws.EnvAuth()
	if err != nil {
		log.Fatal(err)
	}
	e := ec2.New(auth, region)
	resp, err := e.Instances(nil, filter)

	instances := flattenReservations(resp.Reservations)
	targetGroups := groupByTags(instances, tags)
	renderConfig(os.Stdout, targetGroups)
}

func initFlags() {
	var (
		tagsRaw   string
		regionRaw string
	)

	flag.StringVar(&tagsRaw, "tags", "Name", "Comma seperated list of tags to group by (e.g. `Environment,Application`)")
	flag.StringVar(&regionRaw, "region", "us-west-2", "AWS region to query")
	flag.IntVar(&port, "port", 80, "Port that is exposing /metrics")

	flag.Parse()
	tags = strings.Split(tagsRaw, ",")
	region = aws.Regions[regionRaw]
}

func groupByTags(instances []ec2.Instance, tags []string) map[string]*TargetGroup {
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

		target := fmt.Sprintf("%s:%d", instance.PrivateIpAddress, port)
		targetGroup.Targets = append(targetGroup.Targets, target)
	}

	return targetGroups
}

func renderConfig(wr io.Writer, targetGroups map[string]*TargetGroup) {
	// We need to transform targetGroups into a values list sorted by key
	tgList := []*TargetGroup{}
	keys := []string{}
	for k, _ := range targetGroups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		tgList = append(tgList, targetGroups[k])
	}

	b, err := json.MarshalIndent(tgList, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	_, err = wr.Write(b)
	if err != nil {
		log.Fatal(err)
	}
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
