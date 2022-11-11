# deye-logger-at-cmd

Deye based micro inverters use a built-in WLAN module for quick configuration.

The WLAN module works in AP+STA mode, i.e. it starts an access point and is able to connect to another access point at the same time.

Unfortunately, after configuration, neither the inverter's own access point can be disabled, nor can the default password of `12345678` be changed.

So the hurdle to connect to the inverter's own access point is very low.

In addition to the web based configuration, which can be protected by username and password, it is possible to configure the micro inverter via AT commands on port `48899`. For this purpose, a fixed passphrase `WIFIKIT-214028-READ` is used which in principle cannot be changed since it is already used in iOS and Android apps of the ?manufacturer?.

On the one hand, it is possible to access the inverter, i.e. to enter the operator's own WLAN network.

### TL;DR
This tool reads settings for WLAN (incl. connected SSID and WPA key) as well as web login credentials from the logger.

## Supported Devices

Currently, the deye microinverters are affected by this. Also known under other brands like Bosswerk or Turbo Energy. If your device is also affected, please let me know.

## Dependencies

- Golang is required to build

## Usage

Simply type `main` to print the help

`main`

To read out the settings please type the following

`main -t <ip of the logger>:48899`

If you are interested in what is happening here, you can get the communication output

`main -t <ip of the logger>:48899 -xv`

Example output

```
2022/11/01 10:08:25 * Connecting :0 -> <Inverters IP>:48899...
2022/11/01 10:08:25 > WIFIKIT-214028-READ
2022/11/01 10:08:26 < <Inverters IP>,<Inverters MAC>,<Inverters MID>
2022/11/01 10:08:26 > +ok
2022/11/01 10:08:27 > AT+WAP
2022/11/01 10:08:28 < +ok=11BGN,AP_<Inverters MID>,CH1
2022/11/01 10:08:28 > AT+WAKEY
2022/11/01 10:08:29 < +ok=WPA2PSK,AES,12345678
2022/11/01 10:08:29 > AT+WSSSID
2022/11/01 10:08:30 < +ok=<Your SSID>
2022/11/01 10:08:30 > AT+WSKEY
2022/11/01 10:08:31 < +ok=WPA2PSK,AES,<Your WPA key>
2022/11/01 10:08:31 > AT+WANN
2022/11/01 10:08:32 < +ok=DHCP,<Inverters IP>,<Inverters Sbunet>,<Inverters GW>
2022/11/01 10:08:32 > AT+WEBU
2022/11/01 10:08:33 < +ok=<Your configured username>,<Your configured password>
2022/11/01 10:08:33 > AT+Q
2022/11/01 10:08:34 AP settings
2022/11/01 10:08:34     Mode, SSID and Chanel:  11BGN,AP_AP_<Inverters MID>,CH1
2022/11/01 10:08:34     Encryption:             WPA2PSK,AES,12345678
2022/11/01 10:08:34 Station settings
2022/11/01 10:08:34     SSID:                   <Your SSID>
2022/11/01 10:08:34     Key:                    WPA2PSK,AES,<Your WPA key>
2022/11/01 10:08:34     IP:                     DHCP,<Inverters IP>,<Inverters Sbunet>,<Inverters GW>
2022/11/01 10:08:34 Web settings
2022/11/01 10:08:34     Login:                  <Your configured username>,<Your configured password>
```

### Sending AT-Commands

`main -t <ip of the logger>:48899 -xat <at command>`

Example
```
main -t <ip of the logger>:48899 -xat AT+WEBVER
2022/11/11 12:37:51 * Connecting :0 -> <ip of the logger>:48899...
2022/11/11 12:37:54 +ok=V1.0.24
```

### Sending ModBus read command

`main -t <ip of the logger>:48899 -xmb <Start_Register+Length>`

So with a start register address of 0012 and a length of 0001 only one register is read.

```
main -t <ip of the logger>:48899 -xmb 00120001
2022/11/11 12:39:26 * Connecting :0 -> <ip of the logger>:48899...
2022/11/11 12:39:29 +ok=01030204017B44
```

Explanation
```
01      is the slave id
03      is the function code (read)
02      is the length of the payload (2 bytes)
0401    is the playload, i that case 04 is the number of MPPT and 01 is number of ac phases (you need to know how to interpret the register.)
7B44    is the crc16
```
