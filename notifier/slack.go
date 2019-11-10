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
	Message     string       `json:"text"`
	Attachments []attachment `json:"attachments"`
}
type SlackField struct {
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
	Short *bool  `yaml:"short,omitempty" json:"short,omitempty"`
}

type attachment struct {
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Pretext    string       `json:"pretext,omitempty"`
	Text       string       `json:"text"`
	Fallback   string       `json:"fallback"`
	CallbackID string       `json:"callback_id"`
	Fields     []SlackField `json:"fields,omitempty"`
	ImageURL   string       `json:"image_url,omitempty"`
	ThumbURL   string       `json:"thumb_url,omitempty"`
	Footer     string       `json:"footer"`
	Color      string       `json:"color,omitempty"`
	MrkdwnIn   []string     `json:"mrkdwn_in,omitempty"`
}

func NewSlackNotifier(webhook string) *Slack {
	return &Slack{
		Webhook: webhook,
		Client: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func createSlackRequest(victim v1.Pod) request {
	attach := attachment{
		Title:  "Chaos event - Pod termination",
		Text:   fmt.Sprintf("pod %s has been selected by chaos-kube for termination", victim.Name),
		Footer: "chaos-kube",
		Color:  "#F35A00",
	}

	short := len(victim.Namespace) < 20 && len(victim.Name) < 20

	attach.Fields = []SlackField{
		{
			Title: "namespace",
			Value: victim.Namespace,
			Short: &short,
		},
		{
			Title: "pod",
			Value: victim.Name,
			Short: &short,
		},
	}

	return request{
		Attachments: []attachment{attach},
	}
}
func (s Slack) NotifyTermination(victim v1.Pod) error {
	message := createSlackRequest(victim)

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
