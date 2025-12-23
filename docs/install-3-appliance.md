## # Step 5: Make the Raspberry Pi into a quiz appliance
QuizInch supplies the files needed to dedicate a Raspberry Pi as an appliance. That is:
- It starts up quiz operation whenever the device is switched on.
- It can provide a dedicated Wi-Fi network (i.e. an access point) for the other quiz devices.

Use `raspi-config` to select boot to Console instead of Desktop.

Appliance operation requires some packages that are not pre-installed on Raspberry Pi OS: `sudo apt update` and `sudo apt install dialog wtype`.

Because the appliance files change the configuration of the RPi, you must copy them to the appropriate locations yourself, by running a script supplied by QuizInch: `sudo sh /srv/quizinch/setup/appliance.sh`. Or if you prefer, use the supplied files as an example for your own setup.

`appliance.sh` also enables the appliance to host a dedicated Wi-Fi network called QUIZ-RPI.
- The network is enabled whenever the appliance cannot connect to an external Wi-Fi network.
- After setup, change the network password from `quizinch.ap` to a password of your own. Optionally, change the  SSID from QUIZ-RPI to a name of your own.
- After running `appliance.sh`, restart the appliance for the network changes to take effect: `sudo systemctl reboot`.

A User Instructions document is supplied for use with the quiz appliance. You should make these changes to the document:
- Change or write in `[RPi password]` to be the password for the system user (e.g. `pi`), as specified when you installed RPi OS.
- Change or write in `[WiFi password 1]` to be the hosted WiFi password, as above.
- Change or write in `[WiFi password 2]` to be the WiFi password for an external network, as set using `raspi-config` when you installed RPi OS or changed afterwards using the appliance menu.

(You could still use the RPi for other purposes by swapping the Micro SD card for another card with a different copy of RPi OS installed.)