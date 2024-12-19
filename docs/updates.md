# Updates
To pull updated images with fault fixes from Docker Hub:
1. `docker compose pull` to fetch updated images from Docker Hub, without suspending service during the transfer.
1. `docker compose up -d --remove-orphans` to restart the service if there are updated images.
1. `docker image prune` to remove obsolete images.

For new features, check Docker Hub for an `inchworks/quizinch` image tagged `1.0`, `1.1`, `2.0` etc, and edit `docker-compose.yml` to match. A different major version number for PicInch indicates that configuration changes will be needed.

Optionally, set `quiz:image` to `inchworks/quizinch:1` to get the latest minor update when restarting Docker.

For site changes to `configuration.yml` and templates, without an updated image, use `docker compose restart`.

## MariaDB Database
The original example `docker-compose.yml` specified `db:image` as `mariadb:10.4`.
QuizInch needs no updates to work with `mariadb:11.4`, and this version will be supported by the MariaDB Foundation until May 2029.

When upgrading, add `MARIADB_AUTO_UPGRADE` as shown in the current example `docker-compose.yml`, to request internal upgrades to the MariaDB database. This environment setting has no effect once the database has been upgraded.

The example also sets `MARIADB_DISABLE_UPGRADE_BACKUP` assuming that, with the modest size of a gallery database, if there is a problem upgrading it will be easier to initialise a new database and restore from an earlier full backup than attempt to repair database system tables.