package portalsetup

import (
	"errors"
	"fmt"
	"os/exec"
)

const frontendUnit = "xdg-desktop-portal.service"

func FrontendUnitKnown() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func RestartFrontend() error {
	systemctl, err := exec.LookPath("systemctl")
	if err != nil {
		return errors.New("systemctl not found; log out and back in to restart the portal")
	}
	out, err := exec.Command(systemctl, "--user", "restart", frontendUnit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart %s: %w: %s", frontendUnit, err, out)
	}
	return nil
}
