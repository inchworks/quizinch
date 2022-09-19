#!/bin/bash

show_menu () {
    height=0
    width=50
    menu_height=4

    ip_addrs=$(ip -o addr show up primary scope global wlan0 | while read -r num dev fam addr rest; do echo ${addr%/*}; done)
    ssid=$(iw dev wlan0 info | grep ssid | awk '{print $2}')

    # dialog preferred to Debian's whiptail, because it has timeout and allows exit code redefinition.
    # The "3>&1 1>&2 2>&3" mess switches STDOUT and STDERR because dialog sends its output to STDERR :-(.
    # Redefining exit codes to match options.
    opt=$(DIALOG_ERROR=1 \
            DIALOG_ESC=1 \
            DIALOG_CANCEL=5 \
            DIALOG_EXTRA=6 \
            DIALOG_HELP=6 \
            DIALOG_ITEM_HELP=6 \
            dialog \
            --clear \
            --backtitle "$HOSTNAME $ip_addrs $ssid" \
            --title "QuizInch" \
            --nocancel \
            --timeout 30 \
            --menu "Choose one of the following options:" \
            $height $width $menu_height \
            1 "Start Quiz program" \
            2 "Restart with QUIZ-RPI WiFi" \
            3 "Restart and join QUIZ-AP WiFi" \
            4 "Desktop" \
            5 "Console" \
            3>&1 1>&2 2>&3)
    ec=$?
}

sleep 10

while true; do
    clear
    show_menu

    # use error code as option
    if (($ec != 0)); then
        opt=$ec
    fi

    case $opt in
    1)
        # run browser for quiz
        cp ~/dot-xinitrc ~/.xinitrc
        startx
        ;;
    2)
        # enable host WiFi AP
        sudo systemctl unmask hostapd
	    sudo systemctl enable hostapd
	    sudo cp /etc/dhcpcd-hostap.conf /etc/dhcpcd.conf
        sudo systemctl reboot
        ;;
    3)
        # disable host WiFI AP
        sudo cp /etc/dhcpcd-client.conf /etc/dhcpcd.conf
	    sudo systemctl disable hostapd
	    sudo systemctl mask hostapd
        sudo systemctl reboot
        ;;
    4)
        # remove browser and run full desktop
        rm -f ~/.xinitrc
        startx
        ;;
    5)
        # exit to command line console
        clear
        exit
        ;;
    *)
        # ignore
        ;;
    esac
done