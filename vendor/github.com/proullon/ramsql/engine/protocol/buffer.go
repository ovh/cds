package protocol

import (
	"container/list"

	"github.com/proullon/ramsql/engine/log"
)

// UnlimitedRowsChannel buffers incomming message from bufferThis channel and forward them to
// returned channel.
// ONLY CREATED CHANNEL IS CLOSED HERE.
func UnlimitedRowsChannel(bufferThis chan message, firstMessage message) chan []string {
	driverChannel := make(chan []string)
	rowList := list.New()

	rowList.PushBack(firstMessage.Value)

	go func() {
		for {
			// If nothing comes in and nothing is going out...
			if rowList.Len() == 0 && bufferThis == nil {
				close(driverChannel)
				return
			}

			// Create a copy of driverChannel because if there is nothing to send
			// We can disable the case in select with a nil channel and get a chance
			// to fetch new data on bufferThis channel
			driverChannelNullable := driverChannel
			var nextRow []string
			if rowList.Len() != 0 {
				nextRow = rowList.Front().Value.([]string)
			} else {
				driverChannelNullable = nil
			}

			select {
			// In case a new row is sent to driver
			case driverChannelNullable <- nextRow:
				if nextRow != nil {
					rowList.Remove(rowList.Front())
				}
				// In case we receive a new value to buffer from engine channel
			case newRow, ok := <-bufferThis:
				if !ok || newRow.Type == rowEndMessage {
					// Stop listening to bufferThis channel
					bufferThis = nil
					// If there is nothing more to listen and there is nothing in buffer, exit
					// close driverChannel so *Rows knows there is nothing more to read
					if rowList.Len() == 0 {
						if driverChannel != nil {
							close(driverChannel)
						}
						return
					}

					if driverChannel == nil {
						log.Critical("Unlimited: But there is nobody to read it, exiting")
						return
					}
				} else if newRow.Type == errMessage {
					log.Critical("Runtime error: %s", newRow.Value[0])
					if driverChannel != nil {
						close(driverChannel)
					}
					return
				} else {
					// Everything is ok, buffering new value
					rowList.PushBack(newRow.Value)
				}
			case exit := <-driverChannel:
				// this means driverChannel is closed
				// set driverChannel to nil so we don't try to close it again
				driverChannel = nil
				_ = exit
			}
		}
	}()

	return driverChannel
}
