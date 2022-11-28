## # Step 5: Make the Raspberry Pi into a quiz appliance
QuizInch supplies the files needed to dedicate a Raspberry Pi as an appliance. That is:
- It starts up quiz operation whenever the device is switched on.
- It can provide a dedicated WiFi network for the other quiz devices.

Because these files change the behavior of the RPi, you must copy them to the appropriate locations yourself, optionally by running a script supplied by QuizInch: `sudo chmod -R +r /srv/quizinch/setup` and `sudo sh /srv/quizinch/setup/appliance.sh`. Or if you prefer, use the supplied files as an example for your own setup.

If you are using the appliance to host a WiFi network, edit the file `/etc/hostapd/hostapd.conf` to change `wpa_passphrase` to a WiFi password of your own. You might also set `ssid` and change `channel`. One way to edit the file is `sudo nano /etc/hostapd/hostapd.conf`

Appliance operation requires two packages that are not pre-installed on Raspberry Pi OS: `sudo apt update` and `sudo apt install dialog unclutter`.

If you plan to use an external WiFi access point, you could pre-set access when you install Raspberry Pi OS. In this case you will need to use an ethernet connection when you install the quiz system. Or you could use one WiFi network for installation and switch WiFi using `raspi-config` when you deploy the system. Note that, in either case, the settings for hosted WiFi and external WiFi are stored separately.

A User Instructions document is supplied for use with the quiz appliance. You should make these changes to the document:
- Change or write in `[RPi password]` to be the password for the system user (e.g. `pi`), as specified when you installed RPi OS.
- Change or write in `[WiFi password 1]` to be the hosted WiFi password, as above.
- Change or write in `[WiFi password 2]` to be the WiFi password for an external network, as set using `raspi-config` when you installed RPi OS.

(You could still use the RPi for other purposes by swapping the Micro SD card for another card with a different copy of RPi OS installed.)