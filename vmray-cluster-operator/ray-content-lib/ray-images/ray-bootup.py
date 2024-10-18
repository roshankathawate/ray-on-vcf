#!/usr/bin/env python
# -*- coding: utf-8 -*-

# Import necessary modules
import subprocess
from xml.dom.minidom import parseString
from pprint import pprint
import ruamel.yaml

# Command to retrieve OVF environment properties
OVFENV_CMD = "/usr/bin/vmtoolsd --cmd 'info-get guestinfo.ovfEnv'"

# Function to update the image tag in a Docker Compose file
def update_image_tag(compose_file, service_name, new_tag):
    # Load YAML data
    yaml = ruamel.yaml.YAML()
    yaml.indent(mapping=4, sequence=4, offset=2)
    with open(compose_file, 'r') as f:
        data = yaml.load(f)
    # Update image tag for the specified service
    if service_name in data['services']:
        data['services'][service_name]['image'] = f"{data['services'][service_name]['image'].split(':')[0]}:{new_tag}"
    # Write updated data back to the file
    with open(compose_file, 'w') as f:
        yaml.dump(data, f)

# Function to perform Docker login
def docker_login(username, password, registry=""):
    try:
        login_cmd = ['docker', 'login', '--username', username, '--password', password]
        # Include registry if provided
        if registry:
            login_cmd.append(registry)
        # Execute Docker login command
        subprocess.run(login_cmd, check=True)
        print("Docker login successful.")
    except subprocess.CalledProcessError as e:
        print("Error logging in to Docker:", e)

# Function to start Docker Compose stack
def start_docker_compose(compose_file):
    try:
        # Execute Docker Compose up command in detached mode
        subprocess.run(['docker-compose', '-f', compose_file, 'up', '-d'], check=True)
        print("Docker Compose stack started successfully.")
    except subprocess.CalledProcessError as e:
        print("Error starting Docker Compose stack:", e)

# Function to retrieve OVF properties
def get_ovf_properties():
    properties = {}
    # Execute command to retrieve OVF properties
    xml_parts = subprocess.Popen(OVFENV_CMD, shell=True, stdout=subprocess.PIPE).stdout.read()
    raw_data = parseString(xml_parts)
    # Parse XML to extract properties
    for property in raw_data.getElementsByTagName('Property'):
        key, value = property.attributes['oe:key'].value, property.attributes['oe:value'].value
        properties[key] = value
    # Get new Docker image tag and registry credentials from OVF properties
    new_tag = properties.get("Ray_Version", "latest")
    docker_registry = properties.get("Registry_URL", "project-taiga-docker-local.artifactory.vcfd.broadcom.net")
    docker_registry_username = properties.get("Username", "ivelumani")
    docker_registry_password = properties.get("Password", "**")
    # Perform necessary Docker operations
    update_image_tag(compose_file, service_name, new_tag)
    docker_login(docker_registry_username, docker_registry_password, docker_registry)
    start_docker_compose(compose_file)
    # Return retrieved properties
    return properties

# Main program entry point
if __name__ == "__main__":
    # Path to Docker Compose file and service name to update
    compose_file = '/home/ray/ray/docker-compose.yml'
    service_name = 'ray'
    # Retrieve OVF properties and print them
    properties = get_ovf_properties()
    pprint(properties, indent=1, width=80)
    # Write properties to a text file
    with open("/home/ray/ovf_properties.txt", "w") as file:
        pprint(properties, file)
