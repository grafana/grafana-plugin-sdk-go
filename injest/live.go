package injest

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/live"
)

// GrafanaLive connects to grafana server
type GrafanaLive struct {
	client   *live.GrafanaLiveClient
	channels map[string]*live.GrafanaLiveChannel
}

func Connect(url string) (GrafanaLive, error) {
	var err error

	g := GrafanaLive{}

	backend.Logger.Info("Connecting to grafana live: %s", url)
	g.client, err = live.InitGrafanaLiveClient(live.ConnectionInfo{
		URL: url,
	})
	if err != nil {
		return g, err
	}
	g.channels = make(map[string]*live.GrafanaLiveChannel)
	g.client.Log.Info("Connected... waiting for data")
	return g, err
}

func (g *GrafanaLive) getChannel(name string) *live.GrafanaLiveChannel {
	c, ok := g.channels[name]
	if ok {
		return c
	}

	var err error
	addr := live.ChannelAddress{
		Scope:     "grafana",
		Namespace: "broadcast",
		Path:      "telegraf/" + name,
	}
	c, err = g.client.Subscribe(addr)
	if err != nil {
		backend.Logger.Error("error connecting", "addr", addr, "error", err)
	} else {
		backend.Logger.Info("Connected to channel", "addr", addr)
	}
	g.channels[name] = c
	return c
}
