package service

import (
	"errors"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/sirupsen/logrus"
	"icetea/client"
	"icetea/service/subtree"
	"strings"
	"sync"
)

type SubTree interface {
	Sub(packet packets.SubscribePacket) error
	Unsub(packet packets.UnsubscribePacket) error
	Match(topic string) ([]string, error)
	Delete(id string)
}

var Handler = NewHandlerService()

type ClientId = string

type HandlerService struct {
	clients map[ClientId]*client.Client
	mux     sync.RWMutex
}

func NewHandlerService() *HandlerService {
	return &HandlerService{
		clients: make(map[ClientId]*client.Client),
	}
}

func (s *HandlerService) ConnectPacket(client *client.Client, packet *packets.ConnectPacket) error {
	var (
		clientId = packet.ClientIdentifier
		connAck  = packets.NewControlPacket(packets.Connack)
	)
	connAck.(*packets.ConnackPacket).ReturnCode = 0x00
	s.mux.Lock()
	defer s.mux.Unlock()
	if old, ok := s.clients[clientId]; ok {
		old.Close()
	}
	client.SetId(clientId)
	s.clients[clientId] = client
	return client.HandleWrite(connAck)
}

func (s *HandlerService) PublishPacket(client *client.Client, packet *packets.PublishPacket) error {
	var (
		topic      = packet.TopicName
		clients, _ = subtree.GetTopicSub().ReadSubClients(topic)
		errs       []string
	)
	s.mux.RLock()
	defer s.mux.RUnlock()
	for _, c := range clients {
		if conn, ok := s.clients[c.GetID()]; ok {
			newPacket := packet.Copy()
			// TODO: messageId 生成器
			newPacket.MessageID = client.NextMessageId()
			logrus.WithFields(map[string]interface{}{
				"clientId": c.GetID(),
				"packet":   newPacket.Details(),
			}).Debug("publish message to sub client")
			if err := conn.HandleWrite(newPacket); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) != 0 {
		return errors.New(strings.Join(errs, ","))
	}

	return nil
}

func (s *HandlerService) SubscribePacket(client *client.Client, packet *packets.SubscribePacket) error {
	var (
		topics    = packet.Topics
		qoss      = packet.Qoss
		subTopics = topicQosMerge(packet.Topics, packet.Qoss)
		subAck    = packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
		err       error
	)
	subAck.MessageID = packet.MessageID
	err = subtree.GetTopicSub().CreateSub(subTopics, client.GetId(), map[string]string{}, "")
	if err != nil {
		for i := 0; i < len(topics) && i < len(qoss); i++ {
			subAck.ReturnCodes = append(subAck.ReturnCodes, 0x80)
		}
		return client.HandleWrite(subAck)
	}
	for i := 0; i < len(topics) && i < len(qoss); i++ {
		qos := int32(qoss[i])
		subTopics[topics[i]] = qos
		if qos == 1 {
			subAck.ReturnCodes = append(subAck.ReturnCodes, 0x02)
		} else {
			subAck.ReturnCodes = append(subAck.ReturnCodes, 0x00)

		}
	}
	return client.HandleWrite(subAck)
}

func (s *HandlerService) UnsubscribePacket(client *client.Client, packet *packets.UnsubscribePacket) error {
	var (
		unsubAck = packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
		topics   = map[string]int32{}
		clientId = client.GetId()
	)
	for _, v := range packet.Topics {
		topics[v] = 0
	}
	err := subtree.GetTopicSub().DeleteSub(topics, clientId)
	unsubAck.MessageID = packet.MessageID
	if err != nil {
		logrus.WithFields(map[string]interface{}{
			"client_id": clientId,
			"topics":    topics,
		}).Error(`unsub failed `)
		return err
	}
	return client.HandleWrite(unsubAck)
}

func (s *HandlerService) PingPacket(client *client.Client, packet *packets.PingreqPacket) error {
	var (
		pong = packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	)
	return client.HandleWrite(pong)
}

func (s *HandlerService) DisconnectPacket(client *client.Client, packet *packets.DisconnectPacket) error {
	return client.Close()
}

func topicQosMerge(topic []string, qos []byte) map[string]int32 {
	var topics = map[string]int32{}
	for i := 0; i < len(topic) && i < len(qos); i++ {
		topics[topic[i]] = int32(qos[i])
	}
	return topics
}
