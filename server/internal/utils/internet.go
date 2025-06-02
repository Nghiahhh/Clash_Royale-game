package utils

import (
	"fmt"
	"net"
	"time"
)

// CheckAndPrintNetwork checks if the machine has internet connectivity and prints the result.
// Input: none
// Output: prints to the console whether internet is available or not.
func CheckAndPrintNetwork() bool {
	if checkInternet() {
		fmt.Println("✅ Internet connection detected.")
		// printNetworkInfo()
		return true
	} else {
		fmt.Println("❌ No internet connection.")
		return false
	}
}

// checkInternet checks for internet connectivity by DNS lookup and TCP connection.
// Input: none
// Output: returns true if internet is available, false otherwise.
func checkInternet() bool {
	if _, err := net.LookupHost("google.com"); err == nil {
		return true
	}

	if conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second); err == nil {
		conn.Close()
		return true
	}

	return false
}

// printNetworkInfo prints all non-loopback network interfaces and their addresses.
// Input: none
// Output: prints network interface information to the console.
func printNetworkInfo() {
	fmt.Println("Network Interfaces:")
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("  Error getting interfaces:", err)
		return
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		fmt.Printf("\tName: %s\n", iface.Name)
		for _, addr := range addrs {
			fmt.Printf("\t\tAddr: %s\n", addr.String())
		}
	}
}
