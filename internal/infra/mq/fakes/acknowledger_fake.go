package fakes

type FakeAcknowledger struct {
	AckCalls    int
	NackCalls   int
	RejectCalls int

	LastAckTag        uint64
	LastAckMultiple   bool
	LastNackTag       uint64
	LastNackMultiple  bool
	LastNackRequeue   bool
	LastRejectTag     uint64
	LastRejectRequeue bool

	AckErr    error
	NackErr   error
	RejectErr error
}

func (f *FakeAcknowledger) Ack(tag uint64, multiple bool) error {
	f.AckCalls++
	f.LastAckTag = tag
	f.LastAckMultiple = multiple
	return f.AckErr
}

func (f *FakeAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	f.NackCalls++
	f.LastNackTag = tag
	f.LastNackMultiple = multiple
	f.LastNackRequeue = requeue
	return f.NackErr
}

func (f *FakeAcknowledger) Reject(tag uint64, requeue bool) error {
	f.RejectCalls++
	f.LastRejectTag = tag
	f.LastRejectRequeue = requeue
	return f.RejectErr
}
