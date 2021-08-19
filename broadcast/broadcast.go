package broadcast

import (
  "encoding/json"
  "berty.tech/go-orbit-db/iface"
  "context"
  "example.com/m/v2/entry"
  "fmt"
)

func BroadcastRequestDoc(req RequestDoc, topic iface.PubSubTopic, ctx context.Context) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	msg := Message{
		Type: "RequestDoc",
		Payload: string(payload),
	}
  msgBytes, err := json.Marshal(msg)
  if err != nil {
    return err
  }
  fmt.Println(string(msgBytes))
	err = topic.Publish(ctx, msgBytes)
	if err != nil {
		return err
	}
	return nil
}

func BroadcastDoc(e []entry.Entry, topic iface.PubSubTopic, ctx context.Context) error {
	payload, err := json.Marshal(e)
	if err != nil {
		return err
	}
	msg := Message{
		Type: "ResponseDoc",
		Payload: string(payload),
	}
  msgBytes, err := json.Marshal(msg)
  if err != nil {
    return err
  }
	err = topic.Publish(ctx, msgBytes)
	if err != nil {
		return err
	}
	return nil
}
