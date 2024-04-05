package impl

import "fmt"

var (
	SettingErrorLeaseInvalid            = fmt.Errorf("lease must be bigger than 0")
	SettingErrorMaxLockTimeInvalid      = fmt.Errorf("max lock time must be bigger than lease time")
	SettingErrorRetryMinIntervalInvalid = fmt.Errorf("retry min interval must be bigger than 0")
	SettingErrorRetryMaxIntervalInvalid = fmt.Errorf("retry max interval must be bigger than min interval")
)

func failedToLock(err error) error {
	return fmt.Errorf("failed to lock: %v", err.Error())
}

func failedToTryLock(err error) error {
	return fmt.Errorf("failed to try lock: %v", err)
}

func failedToRelease(err error) error {
	return fmt.Errorf("failed to release: %v", err)
}
