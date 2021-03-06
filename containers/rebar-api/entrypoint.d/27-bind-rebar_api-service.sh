#!/bin/bash

if [[ -f $BUILT_CFG_FILE ]]; then
    # Add keys into the system
    bind_service rebar-api_service
    rebar deployments bind system to rebar-access || :
    keys=`jq -r .ssh_keys ${BUILT_CFG_FILE}`
    set_service_attrib rebar-access rebar-access_keys "{ \"value\": $keys }"
    set_service_attrib rebar-access rebar-machine_key "{ \"value\": \"`cat /etc/rebar.install.key`\" }"
fi
