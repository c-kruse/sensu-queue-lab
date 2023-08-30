#!/bin/bash

mkdir -p reports
ansible-inventory -i ansible/inventory --list | jq -rc '._meta.hostvars| with_entries(select(.key | startswith("consumer"))) | map(.ansible_host) | .[]' | while read i
do
    scp ec2-user@$i:\*.out ./reports
done

