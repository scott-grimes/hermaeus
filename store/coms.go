package store
// const DiscoveryInterval = time.Hour
//
// // DiscoveryServiceTag is used in our mDNS advertisements to discover other chat peers.
// const DiscoveryServiceTag = "pubsub-chat-example"
//
//
//
// func JoinChatRoom(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID, nickname string, roomName string) (*ChatRoom, error) {
// 	// join the pubsub topic
// 	topic, err := ps.Join(topicName(roomName))
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// and subscribe to it
// 	sub, err := topic.Subscribe()
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	cr := &ChatRoom{
// 		ctx:      ctx,
// 		ps:       ps,
// 		topic:    topic,
// 		sub:      sub,
// 		self:     selfID,
// 		nick:     nickname,
// 		roomName: roomName,
// 		Messages: make(chan *ChatMessage, ChatRoomBufSize),
// 	}
//
// 	// start reading messages from the subscription in a loop
// 	go cr.readLoop()
// 	return cr, nil
// }
//
//
// func (ui *ChatUI) handleEvents() {
// 	peerRefreshTicker := time.NewTicker(time.Second)
// 	defer peerRefreshTicker.Stop()
//
// 	for {
// 		select {
// 		case input := <-ui.inputCh:
// 			// when the user types in a line, publish it to the chat room and print to the message window
// 			err := ui.cr.Publish(input)
// 			if err != nil {
// 				printErr("publish error: %s", err)
// 			}
// 			ui.displaySelfMessage(input)
//
// 		case m := <-ui.cr.Messages:
// 			// when we receive a message from the chat room, print it to the message window
// 			ui.displayChatMessage(m)
//
// 		case <-peerRefreshTicker.C:
// 			// refresh the list of peers in the chat room periodically
// 			ui.refreshPeers()
//
// 		case <-ui.cr.ctx.Done():
// 			return
//
// 		case <-ui.doneCh:
// 			return
// 		}
// 	}
// }
//
// func (cr *ChatRoom) Publish(message string) error {
// 	m := ChatMessage{
// 		Message:    message,
// 		SenderID:   cr.self.Pretty(),
// 		SenderNick: cr.nick,
// 	}
// 	msgBytes, err := json.Marshal(m)
// 	if err != nil {
// 		return err
// 	}
// 	return cr.topic.Publish(cr.ctx, msgBytes)
// }
//
// func (cr *ChatRoom) readLoop() {
// 	for {
// 		msg, err := cr.sub.Next(cr.ctx)
// 		if err != nil {
// 			close(cr.Messages)
// 			return
// 		}
// 		// only forward messages delivered by others
// 		if msg.ReceivedFrom == cr.self {
// 			continue
// 		}
// 		cm := new(ChatMessage)
// 		err = json.Unmarshal(msg.Data, cm)
// 		if err != nil {
// 			continue
// 		}
// 		// send valid messages onto the Messages channel
// 		cr.Messages <- cm
// 	}
// }
//
//
// h, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
// 	if err != nil {
// 		panic(err)
// 	}
//
//
//   // create a new PubSub service using the GossipSub router
//   	ps, err := pubsub.NewGossipSub(ctx, h)
//   	if err != nil {
//   		panic(err)
//   	}
//
//     err = setupDiscovery(ctx, h)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	// use the nickname from the cli flag, or a default if blank
// 	nick := *nickFlag
// 	if len(nick) == 0 {
// 		nick = defaultNick(h.ID())
// 	}
//
// 	// join the room from the cli flag, or the flag default
// 	room := *roomFlag
//
// 	// join the chat room
// 	cr, err := JoinChatRoom(ctx, ps, h.ID(), nick, room)
// 	if err != nil {
// 		panic(err)
// 	}
// // Set a stream handler on host A. /echo/1.0.0 is
// 	// a user-defined protocol name.
// 	ha.SetStreamHandler("/echo/1.0.0", func(s network.Stream) {
// 		log.Println("listener received new stream")
// 		if err := doEcho(s); err != nil {
// 			log.Println(err)
// 			s.Reset()
// 		} else {
// 			s.Close()
// 		}
// 	})
//
//
//   ha.SetStreamHandler("/echo/1.0.0", func(s network.Stream) {
// 		log.Println("sender received new stream")
// 		if err := doEcho(s); err != nil {
// 			log.Println(err)
// 			s.Reset()
// 		} else {
// 			s.Close()
// 		}
// 	})
