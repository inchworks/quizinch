#!/bin/bash

# Enable the RPi to host a WiFi network
cp /srv/quizinch/setup/dhcpcd-client.conf /etc/dhcpcd-client.conf
cp /srv/quizinch/setup/dhcpcd-hostap.conf /etc/dhcpcd-hostap.conf
cp /srv/quizinch/setup/dnsmasq.conf /etc/dnsmasq.conf
cp /srv/quizinch/setup/hostapd.conf /etc/hostapd/hostapd.conf

# Start the RPi as an appliance with a menu to configure operation
cp /srv/quizinch/setup/dot-bashrc ~/.bashrc
cp /srv/quizinch/setup/dot-xinitrc ~/.xinitrc
cp /srv/quizinch/setup/quiz-menu ~/quiz-menu.sh
