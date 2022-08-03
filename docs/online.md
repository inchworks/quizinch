# Online Version
QuizInch can be re-configured as an online website for a virtual quiz.

It is assumed that the main display for the quiz would be shown to an audience by screen sharing, e.g. using Microsoft Teams or Zoom.

The "online" option adds:
- Website security features, such as HTTPS.
- Access controls for the quiz team, and competitors.

The "remote" option adds:
- Online submission of answers by remote teams.
- Monitoring of team status for the quizmaster.
- Viewing and per-question scoring of answers.

## Warning
These option were developed during a COVID-19 lockdown, but were never used. The code for secure website operation is shared with inchworks/picinch and is likely to be robust. Submission of answers by teams needs careful testing.

If the quiz team members are at different locations, as well as the teams, delays in polling for display synchronisation may become an issue. It might be necessary to increase the "display refresh level" setting to the point where coordination becomes difficult. Again, more testing is needed.

## Setup
These instructions assume a Ubuntu Server host with Docker installed. Other Linux distributions may be similar (but CentOS/RHEL 8 provides a different technology to Docker). A basic installation requires the creation of just two files on the server.

1. Set up a host system with Docker and Docker Compose installed. For example, using a DigitalOcean [Docker Droplet][1].

1. Acquire a domain name, or add a sub-domain to a domain you already own. Set the `A` record for the domain or subdomain to the IP address of your server. This should be done BEFORE starting the QuizInch service.

1. Add `/srv/quizinch/docker-compose.yml`. This Docker Compose file specifies the QuizInch and MariaDB containers to be downloaded from Docker Hub, and the settings to run them on the host system. Use the same Docker Compose file as specified for Laptop and RPi setups, but remove the environment settings for the quiz service.
[&#8658; Docker Setup]({{ site.baseurl }}{% link install-1-docker-compose.md %})

1. Add `/srv/quizinch/configuration.yml`. See the sample [configuration.yml]({{ site.baseurl }}{% link configuration.yml.md %}), which includes all that is necessary for an online server. You must change it to set your own domain name(s) and passwords.

1. In `/srv/quizinch` run `docker compose up -d` When issued the first time, this fetches QuizInch and MariaDB containers from Docker Hub, and starts QuizInch. Then QuizInchInch sets up the database, creates the directories to hold media and certificates (in`/srv/quizinch/`). QuizInch will be restarted automatically when the host system is rebooted.
[&#8658; Commands]({{ site.baseurl }}{% link install-2-commands.md %})

1. Connect to your server by domain name using a web browser and see that you can log in.
[&#8658; Site Administrator]({{ site.baseurl }}{% link administrator.md %})

## Installation steps
These instructions assume a Ubuntu Server host with Docker installed. Other Linux distributions may be similar (but CentOS/RHEL 8 provides a different technology to Docker). A basic installation requires the creation of just two files on the server.

See the sample [configuration.yml]({{ site.baseurl }}{% link configuration.yml.md %}) for 

[1]:	https://marketplace.digitalocean.com/apps/docker