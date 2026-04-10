#import <LocalAuthentication/LocalAuthentication.h>
#import <Foundation/Foundation.h>
#include "touchid.h"

int authenticateTouchID(const char *reason) {
    LAContext *ctx = [[LAContext alloc] init];
    NSError *err = nil;
    if (![ctx canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:&err]) {
        return -1;
    }
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);
    __block int result = 0;
    NSString *reasonStr = [NSString stringWithUTF8String:reason];
    [ctx evaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics
       localizedReason:reasonStr
                reply:^(BOOL success, NSError *error) {
        result = success ? 1 : 0;
        dispatch_semaphore_signal(sem);
    }];
    dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
    return result;
}
