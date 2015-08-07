#!/bin/bash

# Generates a new prometheus.yml and reloads prometheus if the contents
# change. Intended to be run from a cronjob

gen_cmd="$@"
config_path=${PROMETHEUS_CONFIG_PATH:=prometheus.yml}
config_path_new=${config_path}.prospective

set -e
${gen_cmd} > ${config_path_new}
set +e

if [[ $(shasum - < ${config_path}) != $(shasum - < ${config_path_new}) ]]; then
    mv ${config_path} ${config_path}.bak
    mv ${config_path_new} ${config_path}
    logger -s -p info -t prometheus-update-config.sh "Updated ${config_path}, reloading prometheus..."
    pkill -SIGHUP prometheus
else
    rm ${config_path_new}
fi
