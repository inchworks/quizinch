# Raspberry Pi Appliance
(Explain terminal and commands.)

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

You can check that Docker is installed successfully by: `sudo docker run hello-world`.

## Install QuizInch
A basic installation requires the creation of just one file on the server. (Explain terminal.)

1. Create a directory for the quiz server: `mkdir /srv/quizinch`.

1. Add `/srv/quizinch/docker-compose.yml`. This Docker Compose file specifies the QuizInch and MariaDB software to be downloaded from Docker Hub, the settings to run them on the host system, and essential application parameters.
[&#8658; Docker Setup]({{ site.baseurl }}{% link install-1-docker-compose.md %})

1. `cd /srv/quizinch` and run `docker compose up -d` When issued the first time, this fetches QuizInch and MariaDB software from Docker Hub, and starts QuizInch. Then QuizInch sets up the quiz database, and creates the directory to hold media files (`/srv/quizinch/media`). QuizInch will be restarted automatically whenever the host system is rebooted.
[&#8658; Commands]({{ site.baseurl }}{% link install-2-commands.md %})

1. Connect to your server by XXX using a web browser and view the home page for the quiz system.

## Make the Raspberry Pi into a quiz appliance
QuizInch supplies the files needed to dedicate a Raspberry Pi as an appliance. That means it starts up quiz operation whenever the device is switched on. It can also provide dedicated WiFi network for the other quiz devices. (You can still use the RPi for other purposes by swapping the Micro SD card for another one with a different copy of RPi OS installed.)

Each time it is run, QuzInch creates the files needed to configure the quiz appliance in `/src/quizinch/setup`. However, because these files change the behavior of the RPi, you must copy them to the appropriate locations yourself. 

5. These files enable the RPi to host a WiFi network:

```
cd /srv/quizinch/setup
mv etc-dhcpcd-client.conf /etc/dhcpcd-client.conf
mv etc-dhcpcd-hostap.conf /etc/dhcpcd-hostap.conf
mv etc-dnsmasq.conf /etc/dnsmasq.conf
mv etc-hostapd-hostapd.conf /etc/hostapd/hostapd.conf
```

1. These files start the RPi as an appliance with a menu to configure operation.

```sh
cd /srv/quizinch/setup
mv home-dot-bashrc ~/.bashrc
mv home-dot-xinitrc ~/.xinitrc
mv home-quiz-menu ~/quiz-menu.sh
```

## After setup
If needed, you can customize the quiz system:
[&#8658; Customisation]({{ site.baseurl }}{% link install-4-customise.md %})


[1]:	https://www.raspberrypi.com/documentation/computers/getting-started.html#setting-up-your-raspberry-pi
[2]:    [https://docs.docker.com/engine/install/debian/]
