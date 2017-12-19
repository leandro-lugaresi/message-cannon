package events

import (
	"context"

	gendiodes "code.cloudfoundry.org/go-diodes"
)

// manyToOneMessage diode is optiomal for many writes and a single reader
// for Messages.
type manyToOneMessage struct {
	d *gendiodes.Poller
}

// newManyToOneMessage returns a new ManyToOneMessage diode to be used
// with many writers and a single reader.
func newManyToOneMessage(ctx context.Context, size int, alerter gendiodes.Alerter) *manyToOneMessage {
	return &manyToOneMessage{
		d: gendiodes.NewPoller(
			gendiodes.NewManyToOne(size, alerter),
			gendiodes.WithPollingContext(ctx)),
	}
}

// Set inserts the given Message into the diode.
func (d *manyToOneMessage) Set(data Message) {
	d.d.Set(gendiodes.GenericDataType(&data))
}

// Next will return the next Message to be read from the diode. If the
// diode is empty this method will block until a Message is available to be
// read or context is done. In case of context done we will return true on the second return param.
func (d *manyToOneMessage) Next() (Message, bool) {
	data := d.d.Next()
	if data == nil {
		return Message{}, true
	}
	return *(*Message)(data), false
}
