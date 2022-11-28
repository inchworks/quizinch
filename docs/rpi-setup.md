# Raspberry Pi Setup

## Before installation
- Set up a Raspberry Pi 4 or 400 with the current version of Raspberry Pi OS. See [raspberrypi.com][1].

- Install Docker and Docker Compose. Check [docs.docker.com][2] for the latest instructions. At the time of writing, this was the recommended method:
    ```sh
    cd ~/.
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo groupadd docker
    sudo usermod -aG docker $USER
    newgrp docker
    ```

You can check that Docker is installed successfully by: `docker run hello-world`.

## Install QuizInch
A basic installation requires the creation of just one file on the server.

1. Create a directory `/srv/quizinch` for the server: `sudo mkdir /srv` and `cd /srv && sudo install -d -m 0755 -o pi -g pi quizinch`.

1. Add `docker-compose.yml` to the server directory. This Docker Compose file specifies the QuizInch and MariaDB software to be downloaded from Docker Hub, the settings to run them on the host system, and essential application parameters.
[&#8658; Docker Setup]({{ site.baseurl }}{% link install-1-docker-compose.md %})

1. `cd /srv/quizinch` and run `docker compose up -d`. When issued the first time, this fetches QuizInch and MariaDB software from Docker Hub, and starts QuizInch. Then QuizInch sets up the quiz database, and creates the directory to hold media files (`/srv/quizinch/media`). QuizInch will be restarted automatically whenever the RPi appliance is switched on.
[&#8658; Commands]({{ site.baseurl }}{% link install-2-commands.md %})

1. If needed, you can customize the quiz system:
[&#8658; Customisation]({{ site.baseurl }}{% link install-4-customise.md %})

1. Optionally, reconfigure the system with a menu to start the quiz display automatically, and to host a WiFi network.
[&#8658; Make RPi Appliance]({{ site.baseurl }}{% link install-3-appliance.md %})

1. Connect to your server at `http://localhost/` using a web browser and view the home page for the quiz system.

[1]:	https://www.raspberrypi.com/documentation/computers/getting-started.html#setting-up-your-raspberry-pi
[2]:    https://docs.docker.com/engine/install/debian/
