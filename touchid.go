package touchid

/*
#cgo CFLAGS: -x objective-c -fmodules -fblocks
#cgo LDFLAGS: -framework CoreFoundation -framework LocalAuthentication -framework Foundation
#include <stdlib.h>
#include <stdio.h>
#import <LocalAuthentication/LocalAuthentication.h>

#define SUCCESS (0)

static inline void* context_init() {
	return (void*)[[LAContext alloc] init];
}

static inline void context_release(void *context) {
	[(LAContext *)context release];
}

static inline void context_invalidate(void *context) {
	[(LAContext *)context invalidate];
}

static inline int context_evaluate_policy(void *context, enum LAPolicy policy, char const* reason) {
	LAContext *authContext = (LAContext *)context;
	NSError *error;
	if ([authContext canEvaluatePolicy:policy error:&error]) {
		dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);
		__block int ret;
		[authContext evaluatePolicy:policy
			localizedReason:[NSString stringWithUTF8String:reason]
			reply:^(BOOL success, NSError *error) {
				ret = success ? SUCCESS : error.code;
				dispatch_semaphore_signal(semaphore);
			}];
		dispatch_semaphore_wait(semaphore, DISPATCH_TIME_FOREVER);
		dispatch_release(semaphore);
		return ret;
	}
	return error.code;
}
*/
import "C"

import (
	"context"
	"unsafe"
)

//go:generate go run gen/main.go -template=gen/local_authentication.go.tmpl -output=local_authentication_gen.go

func Authenticate(ctx context.Context, policy Policy, reason string) error {
	authContext := C.context_init()
	defer C.context_release(authContext)

	stop := context.AfterFunc(ctx, func() {
		C.context_invalidate(authContext)
	})
	defer stop()

	reasonStr := C.CString(reason)
	defer C.free(unsafe.Pointer(reasonStr))

	code := C.context_evaluate_policy(authContext, C.enum_LAPolicy(policy), reasonStr)
	switch code {
	case C.SUCCESS:
		return nil
	default:
		return errorFromCode(code)
	}
}
