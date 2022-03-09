package twitch

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type TwitchCap string

const (
	Membership TwitchCap = "membership"
	Tags       TwitchCap = "tags"
	Commands   TwitchCap = "commands"
)

const (
	ActionPass       string = "PASS"
	ActionNick       string = "NICK"
	ActionPing       string = "PING"
	ActionPong       string = "PONG"
	ActionJoin       string = "JOIN"
	ActionPart       string = "PART"
	ActionUserstate  string = "USERSTATE"
	ActionRoomstate  string = "ROOMSTATE"
	ActionPrivmsg    string = "PRIVMSG"
	ActionNotice     string = "USERNOTICE"
	ActionCapRequest string = "CAP REQ"
)

const (
	TwitchURL    string = "ws://irc-ws.chat.twitch.tv:80"
	TwitchTLSURL string = "wss://irc-ws.chat.twitch.tv:443"
)

type Client struct {
	*websocket.Conn

	mx *sync.Mutex

	Stop         chan struct{}
	Errors       chan error
	Capabilities []TwitchCap
	Username     string
	Token        string

	UsersInChannel map[string][]Source
}

type ClientConfig struct {
	Username     string
	Token        string
	TLS          bool // TODO
	Capabilities []TwitchCap
}

func New(config ClientConfig) (*Client, error) {
	client := Client{
		Conn: nil,
		mx:   &sync.Mutex{},

		Stop:         make(chan struct{}),
		Errors:       make(chan error),
		Username:     config.Username,
		Token:        config.Token,
		Capabilities: config.Capabilities,

		UsersInChannel: make(map[string][]Source),
	}

	return &client, nil
}

func (c *Client) Connect(shutdown chan struct{}) {
	conn, _, err := websocket.DefaultDialer.Dial(TwitchURL, nil)
	if err != nil {
		panic(err)
	}

	c.Conn = conn

	if err := c.Authenticate(c.Username, c.Token); err != nil {
		c.Errors <- err
		close(shutdown)
		return
	}

	for _, cap := range c.Capabilities {
		if err := c.Send(fmt.Sprintf("CAP REQ :twitch.tv/%s", cap)); err != nil {
			c.Errors <- fmt.Errorf("Error requesting capability '%s': %v", cap, err)
		}
	}

	go func() {
		defer close(shutdown)
		for {
			select {
			case <-c.Stop:
				Logger().Println("Client: Interrupt received")
				return
			default:
				kind, message, err := c.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
						c.Errors <- fmt.Errorf("error reading message: %v", err)
						return
					} else {
						Logger().Println("Connection closed.")
						return
					}
				}

				switch kind {
				case websocket.TextMessage:
					msgs, err := ParseMessage(string(message))
					if err != nil {
						c.Errors <- err
					}

					for _, msg := range msgs {
						if err := c.route(msg); err != nil {
							c.Errors <- fmt.Errorf("Error routing message: %w", err)
						}
					}
				}
			}
		}
	}()
}

func (c *Client) Close(done chan struct{}) error {
	Logger().Println("Closing connection")
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		Logger().Println("write close:", err)
		return err
	}
	select {
	case <-done:
	case <-time.After(time.Second):
	}
	return c.Conn.Close()
}

func (c *Client) Send(message string) error {
	Logger().Printf("> %s", message)
	return c.Conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (c *Client) Authenticate(username, token string) error {
	if err := c.Send(fmt.Sprintf("PASS %s", token)); err != nil {
		return fmt.Errorf("error sending token: %v", err)
	}
	if err := c.Send(fmt.Sprintf("NICK %s", username)); err != nil {
		return fmt.Errorf("error sending username: %v", err)
	}

	return nil
}

func (c *Client) Join(channels ...string) {
	for _, channel := range channels {
		c.Send(fmt.Sprintf("JOIN #%s", channel)) // nolint
	}
}

func (c *Client) handleJoin(newSource Source, channel string) {
	c.mx.Lock()
	defer c.mx.Unlock()

	usersInChannel, ok := c.UsersInChannel[channel]
	if ok {
		found := false
		for _, source := range usersInChannel {
			if source.Nickname == newSource.Nickname {
				found = true
				break
			}
		}
		if !found {
			usersInChannel = append(usersInChannel, newSource)
		}
	} else {
		usersInChannel = []Source{newSource}
	}

	c.UsersInChannel[channel] = usersInChannel
}

func (c *Client) handlePart(parting Source, channel string) {
	c.mx.Lock()
	defer c.mx.Unlock()

	usersInChannel, ok := c.UsersInChannel[channel]
	if !ok {
		return
	}

	newList := make([]Source, 0)
	for _, source := range usersInChannel {
		if source.Nickname != parting.Nickname {
			newList = append(newList, source)
		}
	}

	c.UsersInChannel[channel] = newList
}

func (c *Client) route(message Message) error {
	switch message.Action {
	case ActionPing:
		if err := c.Send("PONG :tmi.twitch.tv"); err != nil {
			return err
		}
	case ActionJoin:
		Logger().WithFields(logrus.Fields{
			"nickname": message.Source.Nickname,
			"channel":  message.Params[0],
		}).Debug("User joined channel")
		c.handleJoin(message.Source, message.Params[0])
	case ActionPart:
		Logger().WithFields(logrus.Fields{
			"nickname": message.Source.Nickname,
			"channel":  message.Params[0],
		}).Debug("User parted channel")
		c.handlePart(message.Source, message.Params[0])
	case ActionRoomstate:
		fields := map[string]interface{}{
			"channel": message.Params[0],
		}
		for k, v := range message.Tags {
			fields[k] = v
		}

		Logger().WithFields(logrus.Fields(fields)).Info("Room state received")
	case ActionUserstate:
		fields := map[string]interface{}{
			"channel": message.Params[0],
		}
		for k, v := range message.Tags {
			fields[k] = v
		}

		Logger().WithFields(logrus.Fields(fields)).Info("User state received")
	case ActionPrivmsg:
		fields := map[string]interface{}{
			"nickname": message.Source.Nickname,
			"channel":  message.Params[0],
			"message":  strings.TrimPrefix(strings.Join(message.Params[1:], " "), ":"),
		}
		for k, v := range message.Tags {
			fields[k] = v
		}

		// c.handlePrivmsg(message)

		Logger().WithFields(logrus.Fields(fields)).Info("Message received")
	case ActionNotice:
		fields := map[string]interface{}{
			"nickname": message.Source.Nickname,
			"channel":  message.Params[0],
			"message":  strings.TrimPrefix(strings.Join(message.Params[1:], " "), ":"),
		}
		for k, v := range message.Tags {
			fields[k] = v
		}
		Logger().WithFields(logrus.Fields(fields)).Info("Notice received")
	default:
		Logger().Println("< " + message.Raw)
	}

	return nil
}
