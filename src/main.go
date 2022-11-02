package main

//https://www.rabbitmq.com/tutorials/tutorial-one-go.html
import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	fLogger      = flag.String("t", "", "The IP and port of the loggers assistant endpoint [10.10.100.254:48899]")
	fSource      = flag.String("xs", "", "Local source address")
	fWifiCfgCode = flag.String("xc", "WIFIKIT-214028-READ", "WiFi configuration code [WIFIKIT-214028-READ or HF-A11ASSISTHREAD]")
	fVerbose     = flag.Bool("xv", false, "Outputs all communication with the logger")

	lAddress, rAddress *net.UDPAddr
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

	apSSID := send(conn, "AT+WAP\n", 1, 5, true)
	apEnc := send(conn, "AT+WAKEY\n", 1, 5, true)

	staSSID := send(conn, "AT+WSSSID\n", 1, 5, true)
	staKey := send(conn, "AT+WSKEY\n", 1, 5, true)
	staIP := send(conn, "AT+WANN\n", 1, 5, true)

	webUser := send(conn, "AT+WEBU\n", 1, 5, true)

	send(conn, "AT+Q\n", 1, 0, false)

	log.Println("AP settings")
	log.Printf("\tMode, SSID and Chanel:  %s", removeAtOk(*apSSID))
	log.Printf("\tEncryption:             %s", removeAtOk(*apEnc))
	log.Println("Station settings")
	log.Printf("\tSSID:                   %s", removeAtOk(*staSSID))
	log.Printf("\tKey:                    %s", removeAtOk(*staKey))
	log.Printf("\tIP:                     %s", removeAtOk(*staIP))
	log.Println("Web settings")
	log.Printf("\tLogin:                  %s", removeAtOk(*webUser))
	log.Println()
}

func print(message string) {
	if *fVerbose {
		log.Println(message)
	}
}
