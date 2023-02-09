package subtree

import "context"

type TopicService struct {
}

func (t *TopicService) Run(ctx context.Context) error {
	GetTopicSub()
	return nil
}

func (t *TopicService) Stop() error {
	return nil
}

func (t *TopicService) Name() string {
	return `topic manager service`
}
