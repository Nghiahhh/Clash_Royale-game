package main

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"runtime"
)

func main() {
	fmt.Println("Hello from Go backend!")
	if checkInternet() {
		fmt.Println("✅ Internet connection detected.")
		printNetworkInfo()
	} else {
		fmt.Println("❌ No internet connection.")
	}
	printSystemInfo()
}

func checkInternet() bool {
	_, err := net.LookupHost("google.com")
	return err == nil
}

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
		fmt.Printf("  Name: %s\n", iface.Name)
		for _, addr := range addrs {
			fmt.Printf("    Addr: %s\n", addr.String())
		}
	}
}

func printSystemInfo() {
	fmt.Println("System Info:")
	fmt.Printf("  OS: %s\n", runtime.GOOS)
	fmt.Printf("  Architecture: %s\n", runtime.GOARCH)
	hostname, err := os.Hostname()
	if err == nil {
		fmt.Printf("  Hostname: %s\n", hostname)
	}
	currentUser, err := user.Current()
	if err == nil {
		fmt.Printf("  User: %s\n", currentUser.Username)
	}
}
