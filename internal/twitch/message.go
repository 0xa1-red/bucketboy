package twitch

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	server string = "tmi.twitch.tv"
)

type Message struct {
	Timestamp time.Time
	Raw       string
	Type      string
	Source    Source
	Action    string
	Params    []string
	Tags      map[string]string
}

func ParseMessage(message string) ([]Message, error) {
	messages := strings.Split(message, "\n")
	response := []Message{}

	for _, msg := range messages {
		if msg == "" {
			continue
		}

		m := Message{
			Timestamp: time.Now(),
			Raw:       msg,
		}

		parts := strings.SplitN(msg, " ", 4)
		index := 0

		if strings.HasPrefix(parts[index], "@") {
			m.Tags = parseTags(parts[index])
			index++
		}

		if index > len(parts) {
			Logger().Println("partial message")
		}

		if strings.HasPrefix(parts[index], ":") {
			source, err := parseSource(parts[index])
			if err != nil {
				return nil, err
			}
			m.Source = source
			index++
		}
		// Logger().Printf("%v", x)

		if index > len(parts) {
			return nil, fmt.Errorf("No command")
		}

		m.Action = parts[index]
		index++

		if index >= len(parts) {
			response = append(response, m)
			continue
		}

		var params []string
		for i, v := range parts[index:] {
			v = strings.Trim(v, "\r")
			if strings.HasPrefix(v, ":") {
				v = strings.Join(parts[index+i:], " ")
				v = strings.TrimPrefix(v, ":")
				params = append(params, v)
				break
			}

			params = append(params, v)
		}

		m.Params = params

		response = append(response, m)
	}

	return response, nil
}

type Source struct {
	Nickname string
	Username string
	Hostname string
}

func parseSource(raw string) (Source, error) {
	var s Source

	raw = strings.TrimPrefix(raw, ":")

	if raw == server {
		return Source{
			Username: raw,
		}, nil
	}

	re, err := regexp.Compile("!|@")
	if err != nil {
		return s, fmt.Errorf("Error compiling regex: %w", err)
	}

	parts := re.Split(raw, -1)
	if len(parts) == 0 {
		return s, nil
	}

	switch len(parts) {
	case 1:
		s.Hostname = parts[0]
	case 2:
		// Getting 2 items extremely rare, but does happen sometimes.
		// https://github.com/gempir/go-twitch-irc/issues/109
		s.Nickname = parts[0]
		s.Hostname = parts[1]
	default:
		s.Nickname = parts[0]
		s.Username = parts[1]
		s.Hostname = parts[2]
	}

	return s, nil
}

func parseTags(raw string) map[string]string {
	raw = strings.TrimPrefix(raw, "@")

	tags := make(map[string]string)
	parts := strings.Split(raw, ";")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		tags[kv[0]] = parseTagValue(kv[1])
	}

	return tags
}

func parseTagValue(rawValue string) string {
	tagEscapeCharacters := []struct {
		from string
		to   string
	}{
		{`\s`, ` `},
		{`\n`, ``},
		{`\r`, ``},
		{`\:`, `;`},
		{`\\`, `\`},
	}
	for _, escape := range tagEscapeCharacters {
		rawValue = strings.ReplaceAll(rawValue, escape.from, escape.to)
	}

	rawValue = strings.TrimSuffix(rawValue, "\\")

	rawValue = strings.TrimSpace(rawValue)

	return rawValue
}
