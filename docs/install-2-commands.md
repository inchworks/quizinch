
# Step 3: Commands

## Docker
`docker compose up -d` When issued the first time, sets up the database, creates the directories to hold media files (in`/srv/quizinch/`), and starts QuizInch. On later invocations, it checks for updates and configuration changes to QuizInch, and restarts it if needed.

`docker compose restart` Restarts QuizInch, reading any changes to `configuration.yml` and site-specific graphics.

`docker compose down` Stops QuizInch.

`docker compose logs --tail=100` View the last e.g. 100 entries in application logs.
Look here for any startup errors.

For new features, check Docker Hub for an `inchworks/quizinch` image tagged `1.0`, `1.1`, `2.0` etc, and edit `docker-compose.yml` to match. A different major version number for QuizInch indicates that configuration changes will be needed.