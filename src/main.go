package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Handler func(conn *net.UDPConn)

var (
	fLogger      = flag.String("t", "", "The IP and port of the loggers assistant endpoint [10.10.100.254:48899]")
	fSource      = flag.String("xs", "", "Local source address")
	fWifiCfgCode = flag.String("xc", "WIFIKIT-214028-READ", "WiFi configuration code [WIFIKIT-214028-READ or HF-A11ASSISTHREAD]")
	fAtCmd       = flag.String("xat", "", "Send AT command instead of credentials")
	fModBus      = flag.String("xmb", "", "Send Modbus read register instead of credentials [00120001] -> Read register = 0x0012, length = 0x0001")
	fModBusW     = flag.String("xmbw", "", "Send Modbus write register instead of credentials [00280001020064] -> Write register = 0x0028, length = 0x0001, value length = 0x02, value = 0x0064")
	fVerbose     = flag.Bool("xv", false, "Outputs all communication with the logger")

	lAddress, rAddress *net.UDPAddr
	handler            Handler = nil
)

func init() {
	flag.Parse()

	var err error

	if *fSource == "" {
		lAddress = nil
	}

	if *fLogger == "" {
		flag.Usage()
		os.Exit(1)
	}

	lAddress, err = net.ResolveUDPAddr("udp4", *fSource)
	if err != nil {
		log.Fatal(err)
	}

	rAddress, err = net.ResolveUDPAddr("udp4", *fLogger)
	if err != nil {
		log.Fatal(err)
	}

	checkForArgConflicts := func() {
		if handler != nil {
			fmt.Println("You can't use xat, xmb or xmbw at the same time")
			fmt.Println()
			flag.Usage()
			os.Exit(1)
		}
	}

	if fAtCmd != nil && *fAtCmd != "" {
		handler = atCommandHandler
	}

	if fModBus != nil && *fModBus != "" {
		checkForArgConflicts()

		if len(*fModBus) != 8 {
			fmt.Println("xmb needs first register address and length")
			fmt.Println("First register 0x0012")
			fmt.Println("Length 0x0001")
			fmt.Println("-> 00120001")
			fmt.Println()
			flag.Usage()
			os.Exit(1)
		}
		handler = modBusReadHandler
	}

	if fModBusW != nil && *fModBusW != "" {
		checkForArgConflicts()

		if len(*fModBusW) < 14 {
			fmt.Println("xmbw needs first register address, length, value length, and value")
			fmt.Println("First register 0x0028")
			fmt.Println("Length 0x0001")
			fmt.Println("Value length 0x02")
			fmt.Println("Value 0x0064")
			fmt.Println("-> 00280001020064")
			fmt.Println()
			flag.Usage()
			os.Exit(1)
		}

		handler = modBusWriteHandler
	}

	if handler == nil {
		handler = credentialsHandler
	}
}

func send(conn net.Conn, message string, pause time.Duration, timeout time.Duration, response bool) *string {
	print(fmt.Sprintf("> %s", strings.TrimSpace(message)))
	_, err := fmt.Fprint(conn, message)

	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(pause * time.Second)

	if response {
		response := receive(conn, timeout)
		return &response
	}

	return nil
}

func receive(conn net.Conn, timeout time.Duration) string {
	conn.SetReadDeadline(time.Now().Add(timeout * time.Second))
	buf := make([]byte, 1500)
	l, err := bufio.NewReader(conn).Read(buf)

	if err != nil {
		log.Fatal(err)
	}

	buf = buf[:l]
	response := strings.TrimSpace(string(buf))

	print(fmt.Sprintf("< %s", response))

	return response
}

func removeAtOk(response string) string {
	return strings.Replace(response, "+ok=", "", 1)
}

func main() {
	log.Printf("* Connecting %s -> %s...", lAddress.String(), rAddress.String())

	conn, err := net.DialUDP("udp", lAddress, rAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	response := send(conn, *fWifiCfgCode, 1, 5, true)
	if response == nil {
		log.Fatal("Empty response from logger")
	}
	send(conn, "+ok", 1, 0, false)

	handler(conn)

	send(conn, "AT+Q\n", 1, 0, false)

	log.Println()
}

func atCommandHandler(conn *net.UDPConn) {
	response := send(conn, fmt.Sprintf("%s\n", *fAtCmd), 1, 5, true)
	log.Println(*response)
}

func modBusReadHandler(conn *net.UDPConn) {
	prefix := "0103" // Slave ID + Function
	cmd := fmt.Sprintf("%s%s", prefix, (*fModBus))
	modBusHandler(conn, prefix, cmd)
}

func modBusWriteHandler(conn *net.UDPConn) {
	prefix := "0110" // Slave ID + Function
	cmd := fmt.Sprintf("%s%s", prefix, (*fModBusW))
	modBusHandler(conn, prefix, cmd)
}

func modBusHandler(conn *net.UDPConn, prefix string, cmd string) {
	data, err := hex.DecodeString(cmd)
	if err != nil {
		log.Fatal(err)
	}
	crc := Modbus(data)
	msg := fmt.Sprintf("%s%s", cmd, hex.EncodeToString(crc))
	msglen := len(msg) / 2

	tresponse := send(conn, fmt.Sprintf("AT+INVDATA=%d,%s\n", msglen, msg), 1, 5, true)

	response := strings.ReplaceAll(*tresponse, string([]byte{0x10}), "")
	log.Println(response)
}

func credentialsHandler(conn *net.UDPConn) {
	apSSID := send(conn, "AT+WAP\n", 1, 5, true)
	apEnc := send(conn, "AT+WAKEY\n", 1, 5, true)

	staSSID := send(conn, "AT+WSSSID\n", 1, 5, true)
	staKey := send(conn, "AT+WSKEY\n", 1, 5, true)
	staIP := send(conn, "AT+WANN\n", 1, 5, true)

	webUser := send(conn, "AT+WEBU\n", 1, 5, true)

	log.Println("AP settings")
	log.Printf("\tMode, SSID and Chanel:  %s", removeAtOk(*apSSID))
	log.Printf("\tEncryption:             %s", removeAtOk(*apEnc))
	log.Println("Station settings")
	log.Printf("\tSSID:                   %s", removeAtOk(*staSSID))
	log.Printf("\tKey:                    %s", removeAtOk(*staKey))
	log.Printf("\tIP:                     %s", removeAtOk(*staIP))
	log.Println("Web settings")
	log.Printf("\tLogin:                  %s", removeAtOk(*webUser))
}

func print(message string) {
	if *fVerbose {
		log.Println(message)
	}
}

// Modbus crc16
const (
	MODBUS uint16 = 0xA001
)

func Modbus(data []byte) []byte {

	var crc uint16 = 0xFFFF

	for _, by := range data {
		crc = crc ^ uint16(by)
		for i := 0; i < 8; i = i + 1 {
			if crc&0x0001 == 0x0001 {
				crc = (crc >> 1) ^ MODBUS
			} else {
				crc = crc >> 1
			}
		}
	}

	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, crc)

	return bs
}
