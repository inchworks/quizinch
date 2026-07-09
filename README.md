<h1 align="center">QuizInch</h1>

<div align="center">
  <h3>
    <a href="https://inchworks.com/info/quizinch">Documentation</a>
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

Configuration files are provided to turn a Raspberry Pi into a quiz appliance that starts automatically and provides a dedicated WiFi network. (Raspberry Pi OS Bullseye is supported but not Bookworm.)

[![Project Status: Moved to https://codeberg.org/inchworks/quizinch – The project has been moved to a new location, and the version at that location should be considered authoritative.](https://www.repostatus.org/badges/latest/moved.svg)](https://www.repostatus.org/#moved) to [https://codeberg.org/inchworks/quizinch](https://codeberg.org/inchworks/quizinch).

Version 1.1 is available on Docker Hub.

For more information, including setup and configuration, see https://inchworks.com/info/quizinch.
