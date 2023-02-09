package client

import (
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/sirupsen/logrus"
	"icetea/pkg"
)

type PacketHandler interface {
	ConnectPacket(client *Client, packet *packets.ConnectPacket) error
	PublishPacket(client *Client, packet *packets.PublishPacket) error
	SubscribePacket(client *Client, packet *packets.SubscribePacket) error
	UnsubscribePacket(client *Client, packet *packets.UnsubscribePacket) error
	PingPacket(client *Client, packet *packets.PingreqPacket) error
	DisconnectPacket(client *Client, packet *packets.DisconnectPacket) error
}

func handlePacket(client *Client, packet packets.ControlPacket, handler PacketHandler) error {
	logrus.WithFields(map[string]interface{}{
		"clientId": client.GetId(),
		"packet": map[string]interface{}{
			"detail": packet.Details(),
			"string": packet.String(),
		},
	}).Debug("handle packet")
	switch packet.(type) {
	case *packets.ConnectPacket:
		return handler.ConnectPacket(client, packet.(*packets.ConnectPacket))
	case *packets.PublishPacket:
		return handler.PublishPacket(client, packet.(*packets.PublishPacket))
	case *packets.SubscribePacket:
		return handler.SubscribePacket(client, packet.(*packets.SubscribePacket))
	case *packets.UnsubscribePacket:
		return handler.UnsubscribePacket(client, packet.(*packets.UnsubscribePacket))
	case *packets.PingreqPacket:
		return handler.PingPacket(client, packet.(*packets.PingreqPacket))
	case *packets.DisconnectPacket:
		return handler.DisconnectPacket(client, packet.(*packets.DisconnectPacket))
	default:
		return pkg.ErrSwitchType
	}
}
