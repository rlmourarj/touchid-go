package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/noamcohen97/touchid-go"
)

func main() {
	timeout := flag.Duration("timeout", 0, "authentication timeout (e.g., 30s)")
	reason := flag.String("reason", "", "reason for authentication (required)")
	policy := flag.String("policy", "device-owner", "policy: 'biometrics' or 'device-owner'")
	flag.Parse()

	if *reason == "" {
		fmt.Fprintln(os.Stderr, "error: -reason is required")
		flag.Usage()
		os.Exit(1)
	}

	var p touchid.Policy
	switch *policy {
	case "biometrics":
		p = touchid.PolicyDeviceOwnerAuthenticationWithBiometrics
	case "device-owner":
		p = touchid.PolicyDeviceOwnerAuthentication
	default:
		log.Fatalf("unknown policy: %s (use 'biometrics' or 'device-owner')", *policy)
	}

	ctx := context.Background()
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	if err := touchid.Authenticate(ctx, p, *reason); err != nil {
		log.Fatalf("authentication failed: %v", err)
	}

	fmt.Println("authentication successful")
}
