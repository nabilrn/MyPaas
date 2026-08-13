package deployment

import "strings"

type projectStreamTopic string

const (
	streamTopicStatus     projectStreamTopic = "status"
	streamTopicMetrics    projectStreamTopic = "metrics"
	streamTopicLogs       projectStreamTopic = "logs"
	streamTopicDeployment projectStreamTopic = "deployment"
)

type projectStreamTopics map[projectStreamTopic]struct{}

func parseProjectStreamTopics(raw string) projectStreamTopics {
	out := make(projectStreamTopics)
	if strings.TrimSpace(raw) == "" {
		out[streamTopicStatus] = struct{}{}
		out[streamTopicMetrics] = struct{}{}
		out[streamTopicLogs] = struct{}{}
		out[streamTopicDeployment] = struct{}{}
		return out
	}
	for _, value := range strings.Split(raw, ",") {
		switch projectStreamTopic(strings.TrimSpace(strings.ToLower(value))) {
		case streamTopicStatus:
			out[streamTopicStatus] = struct{}{}
		case streamTopicMetrics:
			out[streamTopicMetrics] = struct{}{}
		case streamTopicLogs:
			out[streamTopicLogs] = struct{}{}
		case streamTopicDeployment:
			out[streamTopicDeployment] = struct{}{}
		}
	}
	if len(out) == 0 {
		out[streamTopicStatus] = struct{}{}
	}
	return out
}

func (t projectStreamTopics) has(topic projectStreamTopic) bool {
	_, ok := t[topic]
	return ok
}
