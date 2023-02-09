package subtree

import (
	"icetea/service/subtree/proto"
	"sync"
	"time"
)

type SubClient interface {
}

type Client interface {
}

var topicSub *TopicSub
var onceTopicSub sync.Once

// TopicSub 主题订阅状态机
// TODO hash订阅与通配符订阅可以使用两把锁，已提高写的速率
type TopicSub struct {
	mu       *sync.RWMutex
	topicSub *proto.TopicSub
}

func newTreeNode(topicSection, topic string) *proto.TreeNode {
	return &proto.TreeNode{
		TopicSection: topicSection,
		Topic:        topic,
		Clients:      make(map[string]*proto.Client),
		ChildNode:    make(map[string]*proto.TreeNode),
	}
}

// GetTopicSub 获取主题订阅状态机
//
// return: 主题订阅
func GetTopicSub() *TopicSub {
	onceTopicSub.Do(func() {
		topicSub = newTopicSub()
	})
	return topicSub
}

// newTopicSub 创建主题订阅树状态机，对状态机进行初始化
//
// return: 主题订阅状态机
func newTopicSub() *TopicSub {
	t := TopicSub{}
	t.topicSub = newProtoTopicSub()
	t.mu = new(sync.RWMutex)
	return &t
}

func newProtoTopicSub() *proto.TopicSub {

	topicSub := new(proto.TopicSub)

	hash := new(proto.HashSubTopic)
	hash.Clients = make(map[string]*proto.Client)
	hash.HashSubTopic = make(map[string]*proto.TopicClientsID)

	topicSub.Tree = newTreeNode("/", "/")
	topicSub.Hash = hash
	topicSub.Clients = make(map[string]*proto.Client)
	return topicSub
}

// newClient 创建一个客户端
//
// return: 客户端实例
func newClient(clientID string) *proto.Client {

	return &proto.Client{
		ID:        clientID,
		SubTopics: make(map[string]int32),
		Meta:      make(map[string]string),
		Queue:     new(proto.Queue),
	}
}

// ReadClientSubTopics 获取客户端订阅的全部topic,qos
//
// param: clientID 客户端ID
// return: 客户端订阅的topic
func (t *TopicSub) ReadClientSubTopics(clientID string) map[string]int32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	topics := make(map[string]int32)
	if client, ok := t.topicSub.Clients[clientID]; ok {
		topics = client.SubTopics
	}
	return topics
}

// ReadClientInfo 获取客户端详情，包括meta，topic
//
// param: clientID
// return: 客户端实例,是否存在
func (t *TopicSub) ReadClientInfo(clientID string) (*proto.Client, bool) {

	t.mu.RLock()
	defer t.mu.RUnlock()
	if client, ok := t.topicSub.Clients[clientID]; ok {
		return client, ok
	} else {
		return nil, false
	}
}

// ReadSubClients 订阅了topic的所有客户端
//
// param: topic 搜索的主题
// return: 客户端切片，客户端数量
func (t *TopicSub) ReadSubClients(topic string) (clients []*proto.Client, total int) {
	// TODO 分页无法使用map，map的无序性

	t.mu.RLock()
	defer t.mu.RUnlock()
	clients = make([]*proto.Client, 0)
	clientsMap := make(map[string]bool)
	if subClients, ok := t.topicSub.Hash.HashSubTopic[topic]; ok {
		total = len(subClients.Clients)
		for cID := range subClients.Clients {
			if client, ok := t.topicSub.Clients[cID]; ok {
				clients = append(clients, client)
				clientsMap[client.ID] = true
			}
		}
	}
	// 返回通配符订阅的client , 去重处理
	for _, v := range t.readWildcardSubClients(topic) {
		if _, ok := clientsMap[v.GetID()]; !ok {
			clients = append(clients, v)
		}
	}
	return clients, total
}

// readWildcardSubClients 查找订阅了通配符主题的所有客户端
//
// param: topic 带通配符的主题
// return: 客户端切片
func (t *TopicSub) readWildcardSubClients(topic string) (clients []*proto.Client) {

	// FIXME 客户端数量过大时需要分页
	clients = make([]*proto.Client, 0)
	topicSlice := splitTopic(topic)
	topicLevel := len(topicSlice)
	nextParents := []*proto.TreeNode{t.topicSub.Tree}
	for level := 0; level < len(topicSlice); level++ {
		parents := nextParents
		nextParents = make([]*proto.TreeNode, 0)
		for _, parent := range parents {
			if tmp, ok := parent.ChildNode[topicSlice[level]]; ok && tmp.TopicSection == topicSlice[level] {
				nextParents = append(nextParents, tmp)
				if parent == nil {
					return
				}
				if level == topicLevel-1 {
					for _, v := range tmp.Clients {
						clients = append(clients, v)
					}
				} else if level >= topicLevel {
					return clients
				}

			}
			if tmp, ok := parent.ChildNode["+"]; ok {
				nextParents = append(nextParents, tmp)
				if level == topicLevel-1 {
					for _, v := range tmp.Clients {
						clients = append(clients, v)
					}
				} else if level >= topicLevel {
					return clients
				}
			}
			if tmp, ok := parent.ChildNode["#"]; ok {
				for _, v := range tmp.Clients {
					clients = append(clients, v)
				}
			}
		}
	}
	return clients
}

// CreateSub 为一个客户端增加主题订阅
//
// param: topics 订阅的主题
// param: clientID 客户端ID
// param: meta 元数据
// return:
func (t *TopicSub) CreateSub(topics map[string]int32, clientID string, meta map[string]string, nodeIP string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var ok bool
	var c *proto.Client
	if c, ok = t.topicSub.Clients[clientID]; !ok {
		c = newClient(clientID)
	}
	c.Meta = meta
	c.NodeIP = nodeIP
	c.ID = clientID
	t.createClientWithoutLock(c)
	for topic, qos := range topics {
		// 判断topic是否为通配符订阅
		c.SubTopics[topic] = qos
		if !HasWildcard(topic) {
			t.createSimpleTopicWithoutLock(topic, c)
		} else {
			t.createWildcardTopicWithoutLock(topic, c)
		}
	}

	return nil
}

// createClientWithoutLock 创建客户端
//
// param: client
func (t *TopicSub) createClientWithoutLock(client *proto.Client) {
	t.topicSub.Clients[client.ID] = client
}

// createSimpleTopicWithoutLock add simple topic , doesn't have wildcard mark
// createSimpleTopicWithoutLock 为客户端增加一个不含通配符的主题订阅
//
// param: topic 不含通配符的主题
// param: client 客户端实例指针
func (t *TopicSub) createSimpleTopicWithoutLock(topic string, client *proto.Client) {
	if clientsID, ok := t.topicSub.Hash.HashSubTopic[topic]; ok {
		clientsID.Clients[client.GetID()] = time.Now().Unix()
	} else {
		c := new(proto.TopicClientsID)
		c.Clients = make(map[string]int64)
		c.Clients[client.GetID()] = time.Now().Unix()
		t.topicSub.Hash.HashSubTopic[topic] = c
	}
	t.topicSub.Clients[client.GetID()] = client
}

// createWildcardTopicWithoutLock 为客户端增加一条包含通配符的主题, 比如： /a/+/b or /a/#
//
// param: topic 包含通配符的主题
// param: client 客户端实例指针
func (t *TopicSub) createWildcardTopicWithoutLock(topic string, client *proto.Client) {
	topicSlice := splitTopic(topic)
	parent := t.topicSub.Tree
	childTopic := ""
	for level := 0; level <= len(topicSlice); level++ {
		section := topicSlice[level]
		childTopic += "/" + section
		if childNode, ok := parent.ChildNode[section]; ok {
			parent = childNode
		} else {
			childNode = newTreeNode(section, childTopic)
			parent.ChildNode[section] = childNode
			parent = childNode
		}
		if level == len(topicSlice)-1 {
			parent.Clients[client.ID] = client
			break
		}
	}
}

// DeleteSub 为客户端删除主题订阅记录
// param: topics 需要删除的topic
// param: clientID 客户端ID
// return: 是否成功，错误信息
func (t *TopicSub) DeleteSub(topics map[string]int32, clientID string) error {
	t.mu.RLock()
	client, ok := t.topicSub.Clients[clientID]
	t.mu.RUnlock()
	if ok {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.deleteSubWithoutLock(topics, client)
		if len(client.SubTopics) == 0 {
			delete(t.topicSub.Clients, clientID)
		}
	}
	return nil
}

// deleteSubWithoutLock 客户端删除topic,不请求客户端的锁，不可以在并发条件下单独使用。
//
// param: topics 主题
// param: client 客户端实例指针
func (t *TopicSub) deleteSubWithoutLock(topics map[string]int32, client *proto.Client) {

	clientID := client.ID
	for topic := range topics {
		if !HasWildcard(topic) {
			// 从哈希表删除订阅
			t.deleteSimpleSubTopicWithoutLock(topic, clientID)
		} else {
			// 从订阅树中删除订阅
			t.deleteWildcardSubTopicWithoutLock(topic, clientID)
		}
		// 客户端信息中删除订阅了topic
		delete(client.SubTopics, topic)
	}
	// 客户端无订阅的主题时删除客户端记录
	if len(client.SubTopics) == 0 {
		delete(t.topicSub.Clients, clientID)
	}
}

// deleteSimpleSubTopicWithoutLock 删除客户端不包含通配符的topic

// param: topics 主题
// param: clientID 客户端ID
func (t *TopicSub) deleteSimpleSubTopicWithoutLock(topic, clientID string) {

	if tmp, ok := t.topicSub.Hash.HashSubTopic[topic]; ok {
		delete(tmp.Clients, clientID)
		if len(tmp.Clients) == 0 {
			delete(t.topicSub.Hash.HashSubTopic, topic)
		}
	}

}

func (t *TopicSub) deleteWildcardSubTopicWithoutLock(topic, clientID string) {

	topicSlice := splitTopic(topic)
	topicLevel := len(topicSlice)
	parent := t.topicSub.Tree
	for level := 0; level < len(topicSlice); level++ {
		if childNode, ok := parent.ChildNode[topicSlice[level]]; ok {
			if level == topicLevel-1 {
				delete(childNode.Clients, clientID)
				if len(childNode.Clients) == 0 {
					delete(parent.ChildNode, topicSlice[level])
				}
			}
			parent = childNode
			if parent == nil {
				return
			}
		}
	}
}

// DeleteClient delete all sub topics from a client
func (t *TopicSub) DeleteClient(clientID string) error {

	t.mu.RLock()
	if client, ok := t.topicSub.Clients[clientID]; !ok {
		t.mu.RUnlock()
		return nil
	} else {
		t.mu.RUnlock()
		t.mu.Lock()
		t.deleteClientWithoutLock(client)
		t.mu.Unlock()
		return nil
	}
}
func (t *TopicSub) deleteClientWithoutLock(client *proto.Client) {
	t.deleteSubWithoutLock(client.GetSubTopics(), client)
	delete(t.topicSub.Clients, client.ID)
}

func (t *TopicSub) subQosMoreThan0(clientID string) bool {

	if client, ok := t.topicSub.Clients[clientID]; ok {
		return subQosMoreThan0(client.SubTopics)
	}

	return false
}

// updateClientAliveTime 更新客户端活跃时间
//
// param: clientID
// param: timestamp
func (t *TopicSub) updateClientAliveTime(clientID string, timestamp int64) {

	if client, ok := t.topicSub.Clients[clientID]; ok {
		client.AliveTime = timestamp
	}
}
