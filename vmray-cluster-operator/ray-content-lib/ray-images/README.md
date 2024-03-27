## Setting up Ray Service on Ubuntu 22.04

This guide outlines the steps required to set up a Ray service on a virtual machine running Ubuntu 22.04. The service will utilize Docker and Docker Compose to manage containers. Additionally, Python 3 and `ruamel_yaml` will be installed to support the service functionality.

### Prerequisites

- Virtual machine running Ubuntu 22.04
- Access to terminal with sudo privileges

### Step-by-Step Instructions

1. **Update Package Lists:**
    ```
    sudo apt update
    ```

2. **Install Docker and Docker Compose:**
    ```
    sudo apt install docker.io docker-compose
    ```

3. **Install Python 3 and pip:**
    ```
    sudo apt install python3 python3-pip
    ```

4. **Install ruamel_yaml:**
    ```
    pip3 install ruamel_yaml
    ```

5. **Adjust Docker Permissions:**
    ```
    sudo usermod -aG docker $USER
    sudo chown $USER:docker /var/run/docker.sock
    sudo chmod 660 /var/run/docker.sock
    ```

6. **Create the systemd service file `ray-service.service`:**
    ```
    sudo nano /etc/systemd/system/ray-service.service
    ```

    Inside the file, place the necessary configurations for your service. For example:
    ```
    [Unit]
    Description=Run RAY Service
    Wants=docker.service
    After=docker.service

    [Service]
    Type=simple
    ExecStart=python3 /home/ray/ray-bootup.py
    StandardOutput=file:/home/ray/service/stdout.log
    StandardError=file:/home/ray/service/stderr.log


    [Install]
    WantedBy=multi-user.target
    ```

    Replace `<your_username>` with your actual username and adjust the `WorkingDirectory` accordingly.

7. **Reload systemd:**
    ```
    sudo systemctl daemon-reload
    ```

8. **Enable the `ray-service` service:**
    ```
    sudo systemctl enable ray-service
    ```

9. **Check the status of the `ray-service` service:**
    ```
    sudo systemctl status ray-service
    ```

10. **Copy your bootup file to the `ray` root user:**
    ```
    sudo cp ray-bootup.py /home/ray/
    ```

11. **Create the directory structure for Docker Compose:**
    ```
    mkdir -p /home/ray/ray
    ```

12. **Create the `docker-compose.yml` file:**
    ```
    nano /home/ray/ray/docker-compose.yml
    ```

    Inside this file, define your Docker Compose configuration.

12. **Content Library Location**
    ```
    http://10.215.20.2/ray-ubuntu-iso/
    ``` 

Feel free to customize this README with additional information or formatting according to your needs.
