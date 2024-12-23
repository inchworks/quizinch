# Step 2: docker-compose.yml

Copy this example, and save it as `docker-compose.yml` in the server directory.

```yml
version: '3'

services:

  db:
    image: mariadb:11.4
    container_name: quiz_db
    expose:
      - 3306
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: "<root-password>"
      MYSQL_DATABASE: quiz
      MYSQL_USER: server
      MYSQL_PASSWORD: "<server-password>"
      MARIADB_AUTO_UPGRADE: 1
      MARIADB_DISABLE_UPGRADE_BACKUP: 1
    volumes:
      - mysql:/var/lib/mysql
    logging:
      driver: "json-file"
      options:
        max-size: "2m"
        max-file: "5"

  quiz:
    image: inchworks/quizinch:latest
    ports:
      - 80:8000
    restart: always
    environment:
      db-password: "<server-password>"
      options: "RPi"
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - ./media:/media
      - ./setup:/setup
      - ./site:/site:ro
    logging:
      driver: "json-file"
      options:
        max-size: "5m"
        max-file: "5"
    depends_on:
      - db

volumes:
  mysql:
```

Edit the example to change the following items. (Take care to keep indentation unchanged when editing. Do not use tabs.)
- `MY_SQL_ROOT_PASSWORD`
- `MYSQL_PASSWORD` and `db-password` Make them the same.

If you intend to change many other QuizInch configuration settings, you may prefer to omit the environment settings here, and set them in a site/configuration.yml file instead. See [configuration.yml]({{ site.baseurl }}{% link configuration.yml.md %})