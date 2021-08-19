package store

import (
  "berty.tech/go-orbit-db/iface"
  "context"
  "berty.tech/go-orbit-db/pubsub/pubsubcoreapi"

  "berty.tech/go-orbit-db/events"
  "example.com/m/v2/broadcast"
  "example.com/m/v2/event"

  "fmt"
  "encoding/json"

  "go.uber.org/zap"
  "time"
  orbitdb "berty.tech/go-orbit-db"
)

type PubSub struct {
	events.EventEmitter

	Pubsub iface.PubSubInterface
  Logger *zap.Logger
  Store orbitdb.KeyValueStore
}

func (p *PubSub) TopicSubscribe(ctx context.Context, topic string) (iface.PubSubTopic, error){
  return p.Pubsub.TopicSubscribe(ctx, topic)
}

func (s *Store) NewPubSub(substore orbitdb.KeyValueStore) PubSub {
	return PubSub{
		Pubsub:                pubsubcoreapi.NewPubSub(s.Api, s.Node.Identity, time.Second, s.Logger, nil),
    Logger:                s.Logger,
    Store: substore,
	}
}

func (p *PubSub) ListenForRequestDoc(ctx context.Context, topic iface.PubSubTopic, s Store) (error) {
  // todo
  chPeers, err := topic.WatchPeers(ctx)
	if err != nil {
		return err
	}
	chMessages, err := topic.WatchMessages(ctx)
	if err != nil {
		return err
	}
	go func() {
		for e := range chPeers {
			switch evt := e.(type) {
			case *iface.EventPubSubJoin:
				p.Logger.Debug(fmt.Sprintf("peer %s joined from %s self is", evt.Peer.String()))
			case *iface.EventPubSubLeave:
				p.Logger.Debug(fmt.Sprintf("peer %s left from %s self is", evt.Peer.String()))
			default:
				p.Logger.Debug("unhandled peer event, can't match type")
			}
		}
	}()

	go func() {
		for evt := range chMessages {
			p.Logger.Debug("Got pub sub message")
			content := evt.Content
			var m broadcast.Message
			err := json.Unmarshal(content, &m)
			if err != nil {
				p.Logger.Error("unable to unmarshal message", zap.Error(err))
				continue
			}
      // todo, check if on rate limit list
      // todo validate on some
      p.Logger.Info(m.Type)
      p.Logger.Info(m.Payload)
      switch m.Type {
        case "RequestDoc":
          if s.Role == ReplicatorRole {
            dat := s.handleRequestDoc(m.Payload)
            if dat != nil && len(dat) > 0 {
              err := broadcast.BroadcastDoc(dat, topic, ctx)
              if err != nil {
                p.Logger.Error(err.Error())
              }
            }
          }
        default:
          p.Logger.Debug("unhandled channel event, can't match type")
      }
		}
	}()
	return nil
}

func (p *PubSub) ListenForResponseDoc(docId string, ctx context.Context, topic iface.PubSubTopic, s Store) (error) {
  // todo
  chPeers, err := topic.WatchPeers(ctx)
	if err != nil {
		return err
	}
	chMessages, err := topic.WatchMessages(ctx)
	if err != nil {
		return err
	}


	go func() {
		for e := range chPeers {
			switch evt := e.(type) {
			case *iface.EventPubSubJoin:
				p.Logger.Debug(fmt.Sprintf("peer %s joined from %s self is", evt.Peer.String()))
			case *iface.EventPubSubLeave:
				p.Logger.Debug(fmt.Sprintf("peer %s left from %s self is", evt.Peer.String()))
			default:
				p.Logger.Debug("unhandled peer event, can't match type")
			}
		}
	}()

	go func() {
		for evt := range chMessages {
			p.Logger.Debug("Got pub sub message")
			content := evt.Content
			var m broadcast.Message
			err := json.Unmarshal(content, &m)
			if err != nil {
				p.Logger.Error("unable to unmarshal message", zap.Error(err))
				continue
			}
      // todo, check if on rate limit list
      // todo validate on some
      rs := s.GetRootStore()
      p.Logger.Debug("about to enter validation")
      if rs.Store == nil {
        continue
      }
      switch m.Type {
        case "ResponseDoc":
          isValid, err := broadcast.ValidateMessage(m, rs.Store)
    			if err != nil {
    				p.Logger.Error("Unable to validate message", zap.Error(err))
    				continue
    			} else if !isValid {
    				p.Logger.Error("Message is invalid")
    				continue
    			}
          dat := s.handleResponseDoc(m.Payload)
					if dat != nil {
						p.Emit(ctx,event.NewEventResponseDocs(dat))
					} else {
						p.Logger.Debug("ResponseDoc error")
					}

        default:
          p.Logger.Debug("unhandled channel event, can't match type")
      }
		}
	}()
	return nil
}
