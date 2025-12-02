package dnp3

import (
	"avaneesh/dnp3-go/pkg/channel"
	"avaneesh/dnp3-go/pkg/internal/logger"
	"avaneesh/dnp3-go/pkg/master"
)

// newMaster creates a new master instance
func newMaster(config MasterConfig, callbacks MasterCallbacks, ch *channel.Channel, log logger.Logger) (Master, error) {
	return master.New(config, callbacks, ch, log)
}
