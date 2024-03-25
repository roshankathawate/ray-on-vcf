#!/usr/bin/env python
# -*- coding: utf-8 -*-

import subprocess
from xml.dom.minidom import parseString
from pprint import pprint
import ruamel.yaml

OVFENV_CMD = "/usr/bin/vmtoolsd --cmd 'info-get guestinfo.ovfEnv'"

def update_image_tag(compose_file, service_name, new_tag):
    yaml = ruamel.yaml.YAML()
    yaml.indent(mapping=4, sequence=4, offset=2)
    with open(compose_file, 'r') as f:
        data = yaml.load(f)
    if service_name in data['services']:
        data['services'][service_name]['image'] = f"{data['services'][service_name]['image'].split(':')[0]}:{new_tag}"
    with open(compose_file, 'w') as f:
        yaml.dump(data, f)

def docker_login(username, password, registry=""):
    try:
        login_cmd = ['docker', 'login', '--username', username, '--password', password]
        if registry:
            login_cmd.append(registry)
        subprocess.run(login_cmd, check=True)
        print("Docker login successful.")
    except subprocess.CalledProcessError as e:
        print("Error logging in to Docker:", e)

def start_docker_compose(compose_file):
    try:
        subprocess.run(['docker-compose', '-f', compose_file, 'up', '-d'], check=True)
        print("Docker Compose stack started successfully.")
    except subprocess.CalledProcessError as e:
        print("Error starting Docker Compose stack:", e)

def get_ovf_properties():
    properties = {}
    xml_parts = subprocess.Popen(OVFENV_CMD, shell=True, stdout=subprocess.PIPE).stdout.read()
    raw_data = parseString(xml_parts)
    for property in raw_data.getElementsByTagName('Property'):
        key, value = property.attributes['oe:key'].value, property.attributes['oe:value'].value
        properties[key] = value
    new_tag = properties.get("Ray_Version", "latest")
    docker_registry = properties.get("Registry_URL", "project-taiga-docker-local.artifactory.eng.vmware.com")
    docker_registry_username = properties.get("Username", "ivelumani")
    docker_registry_password = properties.get("Password", "**")
    print(new_tag)
    print(compose_file)
    update_image_tag(compose_file, service_name, new_tag)
    docker_login(docker_registry_username, docker_registry_password, docker_registry)
    start_docker_compose(compose_file)

    return properties

if __name__ == "__main__":
    compose_file = '/home/ray/ray/docker-compose.yml'
    service_name = 'ray' 
    properties = get_ovf_properties()
    pprint(properties, indent=1, width=80)
    with open("/home/ray/ovf_properties.txt", "w") as file:
        pprint(properties, file)
