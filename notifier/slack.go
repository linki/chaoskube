package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"k8s.io/api/core/v1"
	"net/http"
	"time"
)

const NotifierSlack = "slack"

var DefaultTimeout time.Duration = 15 * time.Second

type Slack struct {
	Webhook string
	Client  *http.Client
}

type request struct {
	Message string `json:"text"`
}

func NewSlackNotifier(webhook string) *Slack {
	return &Slack{
		Webhook: webhook,
		Client: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func (s Slack) NotifyTermination(victim v1.Pod) error {
	message := request{
		Message: fmt.Sprintf("pod %s/%s is begin terminated.", victim.Namespace, victim.Name),
	}
	messageBody, err := json.Marshal(message)

	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, s.Webhook, bytes.NewBuffer(messageBody))

	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := s.Client.Do(req)

	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d from slack webhook %s", res.StatusCode, s.Webhook)
	}

	return nil
}
