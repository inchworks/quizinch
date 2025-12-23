# Raspberry Pi Setup

## Before installation
- Set up a Raspberry Pi 4, 400, 5 or 500 with Raspberry Pi OS Bookworm 64-bit, or a later version. See [raspberrypi.com][1].

- Install Docker Engine. Check [docs.docker.com][2] for the latest instructions. At the time of writing, the instructions for Debian apply to Raspberry Pi OS 64-bit.

- Enable docker to run without sudo: `sudo usermod -aG docker $USER && newgrp docker`.

You can check that Docker is installed successfully by: `sudo docker run hello-world` and `docker compose version`.

## Install QuizInch
A basic installation requires the creation of just one file on the server.

1. Create a directory `/srv/quizinch` for the server: `sudo install -d -m 0755 -g $USER -o $USER /srv/quizinch`.

1. Add `docker-compose.yml` to the server directory. This Docker Compose file specifies the QuizInch and MariaDB software to be downloaded from Docker Hub, the settings to run them on the host system, and essential application parameters.
[&#8658; Docker Setup]({{ site.baseurl }}{% link install-1-docker-compose.md %})

1. `cd /srv/quizinch` and run `docker compose up -d`. When issued the first time, this fetches QuizInch and MariaDB software from Docker Hub, and starts QuizInch. Then QuizInch sets up the quiz database, and creates the directory to hold media files (`/srv/quizinch/media`). QuizInch will be restarted automatically whenever the RPi appliance is switched on.
[&#8658; Commands]({{ site.baseurl }}{% link install-2-commands.md %})

1. If needed, you can customize the quiz system:
[&#8658; Customisation]({{ site.baseurl }}{% link install-4-customise.md %})

1. Connect to your server at `http://localhost/` using a web browser from the Raspberry Pi OS desktop and view the home page for the quiz system.

1. Optionally, reconfigure the system with a menu to start the quiz display automatically, and to host a WiFi network. After rebooting the QuizInch system menu will show the appliance's IP address, as needed to connect an external web browser.
[&#8658; Make RPi Appliance]({{ site.baseurl }}{% link install-3-appliance.md %})

[1]:	https://www.raspberrypi.com/documentation/computers/getting-started.html#setting-up-your-raspberry-pi
[2]:    https://docs.docker.com/engine/install/debian/
