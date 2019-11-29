package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
)

const NotifierSlack = "slack"

var NotificationColor = "#F35A00"
var DefaultTimeout = 10 * time.Second

type Slack struct {
	Webhook string
	Client  *http.Client
}

type slackMessage struct {
	Message     string       `json:"text"`
	Attachments []attachment `json:"attachments"`
}

type slackField struct {
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
	Fields     []slackField `json:"fields,omitempty"`
	ImageURL   string       `json:"image_url,omitempty"`
	ThumbURL   string       `json:"thumb_url,omitempty"`
	Footer     string       `json:"footer"`
	Color      string       `json:"color,omitempty"`
	MrkdwnIn   []string     `json:"mrkdwn_in,omitempty"`
}

func NewSlackNotifier(webhook string) *Slack {
	return &Slack{
		Webhook: webhook,
		Client:  &http.Client{Timeout: DefaultTimeout},
	}
}

func (s Slack) NotifyPodTermination(pod v1.Pod) error {
	title := "Chaos event - Pod termination"
	text := fmt.Sprintf("pod %s has been selected by chaos-kube for termination", pod.Name)

	short := len(pod.Namespace) < 20 && len(pod.Name) < 20
	fields := []slackField{
		{
			Title: "namespace",
			Value: pod.Namespace,
			Short: &short,
		},
		{
			Title: "pod",
			Value: pod.Name,
			Short: &short,
		},
	}

	message := createSlackRequest(title, text, fields)
	return s.sendSlackMessage(message)
}

func createSlackRequest(title string, text string, fields []slackField) slackMessage {
	return slackMessage{
		Attachments: []attachment{{
			Title:  title,
			Text:   text,
			Footer: "chaos-kube",
			Color:  NotificationColor,
			Fields: fields,
		}},
	}
}

func (s Slack) sendSlackMessage(message slackMessage) error {
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
