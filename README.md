<h1 align="center">QuizInch</h1>

<div align="center">
  <h3>
    <a href="https://quizinch.com">Documentation</a>
    <span> | </span>
    <a href="https://hub.docker.com/r/inchworks/quizinch">Docker Repository</a>
  </h3>
</div>

## Features
QuizInch enables the synchronised presentation of questions, answers and scores for a live quiz. 

At a minimum it needs two computers:
- A laptop or Raspberry Pi connected to the digital projector at a venue, runnning the QuizInch server.
- A laptop to enter scores, running just a web browser and connected to the first computer via WiFi.

Additional displays for different purposes are supported using any devices that have a web browser.  

The server software is written in Go for good performance, and installation is simplified by running it under Docker.
The system does not need an internet connection.

Configuration files are provided to turn a Raspberry Pi into a quiz appliance that starts automatically and provides a dedicated WiFi network. 

[![Project Status: WIP – Initial development is in progress, but there has not yet been a stable, usable release suitable for the public.](https://www.repostatus.org/badges/latest/wip.svg)](https://www.repostatus.org/#wip)
A beta test version is available on Docker Hub.

_It has been used and refined over a number of years to manage a Primary Schools Quiz for a Rotary Club. If you are thinking of using it, I suggest you contact me at support@quizinch.com._

For more information, including setup and configuration, see https://quizinch.com.

## Acknowledgments

Go Packages

- [disintegration/imaging](https://github.com/disintegration/imaging) Image processing.
- [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) MySQL driver.
- [golangcollege/sessions](https://github.com/golangcollege/sessions) HTTP session cookies.
- [ilyakaznacheev/cleanenv](https://github.com/ilyakaznacheev/cleanenv) Read configuration file and environment variables.
- [jmoiron/sqlx](https://github.com/jmoiron/sqlx) SQL library extensions.
- [julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) HTTP request router.
- [justinas/alice](https://github.com/justinas/alice) HTTP middleware chaining.
- [justinas/nosurf](https://github.com/justinas/nosurf) CSRF protection.
- [microcosm-cc/bluemonday](https://github.com/microcosm-cc/bluemonday) HTML sanitizer for user input.

JavaScript Libraries
- [Bootstrap](https://getbootstrap.com) Toolkit for responsive web pages.
- [deck.js](http://imakewebthings.com/deck.js/) HTML slideshow.
- [jQuery](https://jquery.com) For easier DOM processing and Ajax.
- [Popper](https://popper.js.org) Tooltip and popover positioning (used by Bootstrap).

Video processing uses [FFmpeg](https://ffmpeg.org).