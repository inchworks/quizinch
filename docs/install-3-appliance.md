## # Step 2: Make the Raspberry Pi into a quiz appliance
QuizInch supplies the files needed to dedicate a Raspberry Pi as an appliance. That is:
- It starts up quiz operation whenever the device is switched on.
- It can provide a dedicated WiFi network for the other quiz devices.

Because these files change the behavior of the RPi, you must copy them to the appropriate locations yourself, by running a script supplied by QuizInch: `sudo sh /srv/quizinch/setup/appliance.sh`.

(You could still use the RPi for other purposes by swapping the Micro SD card for another one with a different copy of RPi OS installed.)