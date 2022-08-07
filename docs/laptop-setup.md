# Laptop Setup
QuizInch runs as a web server. Installation is similar for Windows, MacOS (Apple silicon or Intel), or Linux.

## Before installation
Download and install Docker Desktop. See [docs.docker.com][1] for instructions.

## Install QuizInch
A basic installation requires the addition of just one file on the server.

1. Create a directory `quizinch` for the quiz server.  On MacOS you might `mkdir ~/quizinch && cd ~/quizinch`, and on Windows `md C:\%HOMEPATH%\QuizInch && cd C:\%HOMEPATH%\QuizInch`.

1. Add `docker-compose.yml` to the server directory. This Docker Compose file specifies the QuizInch and MariaDB software to be downloaded from Docker Hub, the settings to run them on the host system, and essential application parameters.
[&#8658; Docker Setup]({{ site.baseurl }}{% link install-1-docker-compose.md %})

1. Run `docker compose up -d` When issued the first time, this fetches QuizInch and MariaDB software from Docker Hub, and starts QuizInch. Then QuizInch sets up the quiz database, and creates the directory to hold media files (`quizinch/media`). QuizInch will be restarted automatically whenever the host system is rebooted.
[&#8658; Commands]({{ site.baseurl }}{% link install-2-commands.md %})

1. If needed, you can customize the quiz system:
[&#8658; Customisation]({{ site.baseurl }}{% link install-4-customise.md %})

1. Connect to your server at `http://localhost/` using a web browser and view the home page for the quiz system.

[1]:    https://docs.docker.com/desktop/
