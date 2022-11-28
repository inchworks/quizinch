#!/bin/bash

# Enable the RPi to host a WiFi network
sudo cp /srv/quizinch/setup/dhcpcd-client.conf /etc/dhcpcd-client.conf
sudo cp /srv/quizinch/setup/dhcpcd-hostap.conf /etc/dhcpcd-hostap.conf
sudo cp /srv/quizinch/setup/dnsmasq.conf /etc/dnsmasq.conf
sudo mkdir -p /etc/hostapd
sudo cp /srv/quizinch/setup/hostapd.conf /etc/hostapd/hostapd.conf

# Start the RPi as an appliance with a menu to configure operation
cp /srv/quizinch/setup/dot-bashrc ~/.bashrc
cp /srv/quizinch/setup/dot-xinitrc ~/dot-xinitrc
cp /srv/quizinch/setup/quiz-menu.sh ~/quiz-menu.sh
chmod +x ~/quiz-menu.sh
