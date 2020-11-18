package mutagen

import (
	"fmt"
	"github.com/satori/go.uuid"
	"nocalhost/pkg/nhctl/tools"
)

func FileSync(folder string, remoteFolder string, port string) error {
	id := uuid.NewV4()
	idStr := id.String()
	_, err := tools.ExecCommand(nil, true, "mutagen", "sync", "create", "--sync-mode=one-way-safe", fmt.Sprintf("--name=nocalhost-%s", idStr), folder, fmt.Sprintf("root@shared-container:%s:%s", port, remoteFolder))
	return err
}
