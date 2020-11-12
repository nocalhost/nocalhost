package cmds

import "testing"

func TestApplication_StopAllPortForward(t *testing.T) {
	application, err := NewApplication("eeee")
	if err != nil {
		printlnErr("fail to create application", err)
		return
	}

	err = application.StopAllPortForward()
	if err != nil {
		printlnErr("fail to stop port-forward", err)
	}
}
