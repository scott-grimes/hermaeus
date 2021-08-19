package event
import (
	"example.com/m/v2/entry"
)

type EventResponseDocs struct {
	Entries []entry.Entry
}

func NewEventResponseDocs(res []entry.Entry) *EventResponseDocs {
	return &EventResponseDocs{
		Entries: res,
	}
}
