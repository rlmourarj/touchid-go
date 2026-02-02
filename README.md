# touchid-go

Go wrapper for Apple's LocalAuthentication framework. Authenticate users with Touch ID on macOS.

## Features

- **Context-aware** - Full support for Go contexts with cancellation and timeouts
- **Policies** - Support many authentication policies
- **Verbose errors** - Distinct sentinel errors for precise error handling
- **Code generation** - Policies and errors auto-generated from Apple headers

## Installation

```bash
go get github.com/noamcohen97/touchid-go
```

## Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/noamcohen97/touchid-go"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    err := touchid.Authenticate(ctx, touchid.PolicyDeviceOwnerAuthentication, "Access your vault")
    if err != nil {
        log.Fatal(err)
    }
}
```
