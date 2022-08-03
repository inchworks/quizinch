# QuizInch
This application enables the synchronised presentation of questions, answers and scores for a live quiz. 

At a minimum it needs two computers:
- A laptop or Raspberry Pi connected to the digital projector at a venue, runnning the QuizInch server.
- A laptop to enter scores, running just a web browser and connected to the first computer via WiFi.

Additional displays for different purposes are supported using any devices that have a web browser.  

The server software is written in Go for good performance, and installation is simplified by running it under Docker.
The system does not need an internet connection.

Configuration files are provided to turn a Raspberry Pi into a quiz appliance that starts automatically and provides a dedicated WiFi network. 

_It has been used and refined over a number of years to manage a Primary Schools Quiz for a Rotary Club. If you are thinking of using it, I suggest you contact me at support@quizinch.com._
