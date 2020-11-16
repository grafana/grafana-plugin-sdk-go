package live

import (
	"fmt"
	"log"
	"time"

	"github.com/centrifugal/centrifuge-go"
)

// GrafanaLiveClient connects to the GrafanaLive server
type GrafanaLiveClient struct {
	connected bool
	client    *centrifuge.Client
	lastWarn  time.Time
	channels  map[string]*GrafanaLiveChannel
}

// GrafanaLiveChannel allows access to a channel within the server
type GrafanaLiveChannel struct {
	id         string
	subscribed bool
	client     *GrafanaLiveClient
	sub        *centrifuge.Subscription
}

// Subscribe will subscribe to a channel within the server
func (c *GrafanaLiveClient) Subscribe(addr ChannelAddress) (*GrafanaLiveChannel, error) {
	id := addr.ToChannelID()
	if !addr.IsValid() {
		return nil, fmt.Errorf("invalid channel: %s", id)
	}

	sub, err := c.client.NewSubscription(id)
	if err != nil {
		return nil, err
	}

	ch := &GrafanaLiveChannel{
		id:     id,
		client: c,
		sub:    sub,
	}

	sub.OnSubscribeSuccess(ch)
	sub.OnSubscribeError(ch)
	sub.OnUnsubscribe(ch)

	err = sub.Subscribe()
	if err != nil {
		return nil, err
	}
	return ch, nil
}

// Publish sends the data to the channel
func (c *GrafanaLiveChannel) Publish(data []byte) {
	if !c.client.connected {
		if time.Since(c.client.lastWarn) > time.Second*5 {
			c.client.lastWarn = time.Now()
			log.Printf("grafana live channel not connected: %s\n", c.id)
		}
		return
	}
	_, err := c.sub.Publish(data)
	if err != nil {
		log.Printf("error publishing: %v", err)
	}
}

//--------------------------------------------------------------------------------------
// CLIENT
//--------------------------------------------------------------------------------------

type liveClientHandler struct {
	client *GrafanaLiveClient
}

func (h *liveClientHandler) OnConnect(c *centrifuge.Client, e centrifuge.ConnectEvent) {
	log.Printf("Connected to grafana live with ID %s", e.ClientID)
	h.client.connected = true
}

func (h *liveClientHandler) OnError(c *centrifuge.Client, e centrifuge.ErrorEvent) {
	log.Printf("Error: %s", e.Message)
}

func (h *liveClientHandler) OnDisconnect(c *centrifuge.Client, e centrifuge.DisconnectEvent) {
	log.Printf("Disconnected from grafana live: %s", e.Reason)
	h.client.connected = false
}

//--------------------------------------------------------------------------------------
// Channel
//--------------------------------------------------------------------------------------

// OnSubscribeSuccess is called when the channel is subscribed
func (c *GrafanaLiveChannel) OnSubscribeSuccess(sub *centrifuge.Subscription, e centrifuge.SubscribeSuccessEvent) {
	c.subscribed = true
	log.Printf("Subscribed on channel %s, resubscribed: %v, recovered: %v", sub.Channel(), e.Resubscribed, e.Recovered)
}

// OnSubscribeError is called when the channel has an error
func (c *GrafanaLiveChannel) OnSubscribeError(sub *centrifuge.Subscription, e centrifuge.SubscribeErrorEvent) {
	c.subscribed = false
	log.Printf("Subscribed on channel %s failed, error: %s", sub.Channel(), e.Error)
}

// OnUnsubscribe is called when the channel is unsubscribed
func (c *GrafanaLiveChannel) OnUnsubscribe(sub *centrifuge.Subscription, e centrifuge.UnsubscribeEvent) {
	c.subscribed = false
	log.Printf("Unsubscribed from channel %s", sub.Channel())
}

// InitGrafanaLiveClient starts a chat server
func InitGrafanaLiveClient(conn ConnectionInfo) (*GrafanaLiveClient, error) {
	url, err := conn.ToWebSocketURL()
	if err != nil {
		return nil, err
	}
	log.Printf("Connecting to %s\n", url)

	c := centrifuge.New(url, centrifuge.DefaultConfig())

	glc := &GrafanaLiveClient{
		client:   c,
		channels: make(map[string]*GrafanaLiveChannel),
	}
	handler := &liveClientHandler{
		client: glc,
	}
	c.OnConnect(handler)
	c.OnError(handler)
	c.OnDisconnect(handler)

	err = c.Connect()
	if err != nil {
		return nil, err
	}

	return glc, nil
}
