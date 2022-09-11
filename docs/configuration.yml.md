# configuration.yml
Configuration parameters may be specified in this file, or as environment variables. Settings here will be overridden by environment variables in docker-compose.yml.
This an example configuration file with just the essential settings for an online website.  

```yml
# Example configuration for online QuizInch server.
#  - Edit and rename to configuration.yml
#  - Take care to keep indentation unchanged when editing. Do not use tabs.

# Enables online server operation and remote competitors.
options:
 - online
 - remote

# This should match the one specified for MYSQL_PASSWORD in docker-compose.yml.
db-password: <server password>

# The following is needed for certificate registration with Let's Encrypt
domains:
  - our-domain.com
  - www.our-domain.com

# Address to be notified of problems with certificates
certificate-email: you@example.com

# A random 32 character key used to encrypt users session data
# For example, start with this one and change a lot of the individual characters.
session-secret: Hk4TEiDgq8JaCNR?WaPeWBf4QQYNUjMR

# Administrator, to be added to the database
admin-name: admin@example.com
admin-password: <your-password>
```

Set the following items as needed. Default values are as shown.
## Database
A database connection is requested with DSN `db-user:db-password@db-source?parseTime=true `. A MariaDB or MySQL database is required.

**db-source** `tcp(quiz_db:3306)/quizinch`

**db-user** `server`

**db-password** `<server-password>`

## Domains
**domains** List of domains for which Let’s Encrypt certificates will be requested on first access.
- The website must be reachable for each specified domain via a DNS entry. 
- The domains are listed one per line, each preceded by `" - "` as shown in the example above.
- The first domain listed will be identified as canonical in page headers.
- If no domains are specified, the website can be accessed as an insecure HTTP server.

This is intended for testing and is not recommended for production.

**certificate-email** Address given to Let’s Encrypt, for notification of problems with certificates.

## Session
**session-secret** A random 32 character key used to encrypt users session data.

## Administrator
Specifies the username and password for a QuizInch administrator if the username does not exist in the database. These items may be removed after setup if desired.

**admin-name** E.g. me@mydomain.com.

**admin-password** `<your-password>`

## Maximum image sizes
Photos uploaded are resized to fit these dimensions.

**image-width**  `1600` stored image width

**image-height** `1200` stored image height

**thumbnail-width** `278` thumbnail width

**thumbnail-height** `208` thumbnail height

**max-upload** `512` maximum image or video upload, in megabytes

## Operational settings
**max-upload-age** `1h` time limit to save a round update, after uploading images. Units m(inutes) or h(ours).

**monitor-interval** `5000` monitor display update (mS)

**slide-items** `10` default maximum items per slide

**thumbnail-refresh** `1h` refresh interval for topic thumbnails. Units m(inutes) or h(ours).

**usage-anon** `1` anonymisation of user IDs: 0 = daily, 1 = immediate.

## Website variants
Settings to change the operation of the website.
**options** ` `
- "online" enables HTTPS, user passwords and other security features.
- "remote" enables online entry of answers by remote teams, and scoring of those answers. "
- "RPi" creates the files needed to dedicate a Raspberry Pi as a quiz appliance.

**home-switch** switches the home page to a specified template, for example, `disabled` to show `disabled.page.tmpl` when the website is offline.

**misc-name** `misc` path in URL for miscellaneous files, as in `example.com/misc/file`

**audio-types** `.mp3,.aac,.flac,.m4a` acceptable audio file types

**video-snapshot** `3s` time within video for snapshot thumbnail. Units s(econds), -ve for no snapshots.

**video-types** `.mp4,.mov` acceptable video file types

## For testing
**test-self** `false` Set "true" to disable online security features that prevent self testing. (The actual tests are run using `go test`.)
