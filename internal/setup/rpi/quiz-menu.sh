#!/bin/bash

# The "3>&1 1>&2 2>&3 3>&-" mess switches stdout (1) and stderr (2) using a temporary file descriptor (3)
# before closing 3. It is needed because dialog sends its output to stderr :-( and we'd like it as stdout.

get_network () {
    ssid=$(iw dev wlan0 info | grep ssid | awk '{print $2}')
    ip_addrs=$(ip -o addr show up primary scope global wlan0 | while read -r num dev fam addr rest; do echo ${addr%/*}; done)
}

set_connection () {
    # get current settings for connection
    ssid1=$(nmcli -f ssid con show $con)
    pw1=$(nmcli -f wifi-sec.psk con show $con)

    values=$(dialog \
        --separate-widget $'\n' \
        --title "$dlg_title" \
        --form "" \
        0 0 0 \
        "Name (SSID):" 1 1 "$ssid1" 1 10 30 0 \
        "Password:" 2 1 "$pw1" 2 10 30 0  \
        3>&1 1>&2 2>&3 3>&-)

    if (($values != "")); then
        # values from dialog
        ssid1=$(echo "$values" | sed -n 1p)
        pw1=$(echo "$values" | sed -n 2p)

        # change settings
        nmcli con modify $con ssid "$ssid1" wifi-sec.psk "$pw1"
        nmcli reload
    fi
}

show_menu () {
    height=0
    width=50
    menu_height=5

    get_network

    # dialog preferred to Debian's whiptail, because it has timeout and allows exit code redefinition.
    # Redefining exit codes to match options.
    opt=$(DIALOG_ERROR=1 \
            DIALOG_ESC=1 \
            DIALOG_CANCEL=6 \
            DIALOG_EXTRA=7 \
            DIALOG_HELP=7 \
            DIALOG_ITEM_HELP=7 \
            dialog \
            --clear \
            --backtitle "WiFi : $ssid, $HOSTNAME website : $ip_addrs" \
            --title "QuizInch" \
            --nocancel \
            --timeout 30 \
            --menu "Choose one of the following options:" \
            $height $width $menu_height \
            1 "Start Quiz program" \
            2 "Show network address" \
            3 "Select external Wi-Fi access" \
            4 "Set Quiz RPi hosted Wi-Fi" \
            5 "Restart Wi-Fi"
            6 "Desktop" \
            7 "Console" \
            3>&1 1>&2 2>&3 3>&-)
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
        # show network address
        get_network
        clear
        echo ""
        echo ""
        echo ""
        echo "        WiFi :" $ssid
        echo "       " $HOSTNAME "website :" $ip_addrs
        echo ""
        read -n 1 -s -r -p "        Press any key to continue"
        ;;
    3)
        # select external network access
        dlg_title="External Network"
        con="preconfigured"
        set_connection
        ;;
    4)
        # set RPi AP network access
        dlg_title="Quiz RPi Network"
        con="access-point"
        set_connection
        ;;
    5)
        # restart Wi-Fi
        nmcli reload
        clear
        echo ""
        echo ""
        echo ""
        echo "        Restarting Wi-Fi"
        echo ""
        read -n 1 -s -r -p "        Press any key to continue"
        ;;
    6)
        # remove browser and run full desktop
        rm -f ~/.xinitrc
        startx
        ;;
    7)
        # exit to command line console
        clear
        exit
        ;;
    *)
        # ignore
        ;;
    esac
done
