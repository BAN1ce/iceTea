package client

import (
	"context"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"sync"
)

type Client struct {
	conn      net.Conn
	Id        string
	handler   PacketHandler
	messageId uint16
	mux       sync.RWMutex
}

func NewClient(conn net.Conn, handler PacketHandler) *Client {

	return &Client{
		conn:      conn,
		handler:   handler,
		messageId: uint16(rand.Intn(10000)),
	}
}

func (c *Client) Run(ctx context.Context) error {
	go c.handleReadByte(ctx)
	return nil
}

func (c *Client) Stop() error {
	return c.Close()
}

func (c *Client) Name() string {
	return `client:` + c.Id
}

func (c *Client) handleReadByte(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:

			if mqttPacket, err := packets.ReadPacket(c.conn); err != nil {
				// TODO:
				c.Close()
				return
			} else {
				if err := handlePacket(c, mqttPacket, c.handler); err != nil {
					// TODO: handle error
					c.Close()
					return
				}
			}
		}
	}

}

func (c *Client) HandleWrite(packet packets.ControlPacket) error {
	logrus.WithField("packet", packet.String()).WithField("detail", packet.Details()).Debug("client.HandleWrite")
	return packet.Write(c.conn)
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetConn() net.Conn {
	return c.conn
}
func (c *Client) GetId() string {
	return c.Id
}

func (c *Client) SetId(id string) {
	c.Id = id
}

func (c *Client) NextMessageId() uint16 {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.messageId += 1
	if c.messageId == uint16(0xff) {
		c.messageId = 0
	}
	return c.messageId
}
