#!/bin/bash

# Use external Wi-Fi network if available
sudo nmcli con modify preconfigured connection.autoconnect-priority 10

# Host Wi-Fi external access point
nmcli con delete access-point
nmcli con add type wifi ifname wlan0 mode ap con-name access-point ssid QUIZ-RPI
nmcli con modify access-point wifi.band bg
nmcli con modify access-point wifi-sec.key-mgmt wpa-psk wifi-sec.psk "quizinch.ap"
nmcli con modify access-point ipv4.method shared ipv4.address 192.168.4.1/24 ipv4.gateway 192.168.4.1
nmcli con modify access-point ipv6.method disabled

# Require WPA2
sudo nmcli con modify access-point \
    802-11-wireless-security.proto rsn \
    802-11-wireless-security.group ccmp \
    802-11-wireless-security.pairwise ccmp
nmcli con up access-point

# Start the RPi as an appliance with a menu to configure operation
cp /srv/quizinch/setup/dot-bashrc ~/.bashrc
cp /srv/quizinch/setup/dot-xinitrc ~/dot-xinitrc
cp /srv/quizinch/setup/quiz-menu.sh ~/quiz-menu.sh
chmod +x ~/quiz-menu.sh
