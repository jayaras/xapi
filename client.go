package xapi

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/c0mm4nd/go-jsonrpc2"
	"github.com/c0mm4nd/go-jsonrpc2/jsonrpc2ws"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
)

const (
	credPrefix            = "auth-"
	credHeader            = "Sec-WebSocket-Protocol"
	textField             = "Text"
	titleField            = "Title"
	durationField         = "Duration"
	optionNumberOfChoices = 5
)

// Command is a JsonRPC2 Method.  Currently this is exposed in the event that we
// do not have everything defined in here and we can set these... though there is
// no way to actually call them directly.
type Command string

const (
	widgetSetValueCommand Command = "xCommand/UserInterface/Extensions/Widget/SetValue"
	alertCommand          Command = "xCommand/UserInterface/Message/Alert/Display"
	promptCommand         Command = "xCommand/UserInterface/Message/Prompt/Display"
	textInputCommand      Command = "xCommand/UserInterface/Message/TextInput/Display"
	ratingCommand         Command = "xCommand/UserInterface/Message/Rating/Display"
	textLineCommand       Command = "xCommand/UserInterface/Message/TextLine/Display"
	muteCommand           Command = "xCommand/Audio/Microphones/Mute"
	unmuteCommand         Command = "xCommand/Audio/Microphones/Unmute"
	feedbackSusbscribe    Command = "xFeedback/Subscribe"
	feedbackUnsubscribe   Command = "xFeedback/Unsubscribe"
	getCommand            Command = "xGet"
)

type (
	// CallbackFunc is the func signature for callbacks for events.
	CallbackFunc func(data []interface{})
	// TextInputOption is a func signature for modifying all the various
	// options you can set for the TextInput UI.
	TextInputOption func(map[string]interface{})
	// TextInputType is the more domain specific element to prompt a
	// user for such as PIN or Password etc.  It helps decide which KeyBoard
	// to show as well as if input should be masked or not.
	TextInputType string
)

const (
	// SingleLine TextInput type.
	SingleLine TextInputType = "SingleLine"
	// Numeric TextInput type, brings up numeric friendly keyboard.
	Numeric TextInputType = "Numeric"
	// Password TextInput type.  Masked input with alphanumeric keyboard.
	Password TextInputType = "Password"
	// PIN TextInput type.  Masked input with numeric friendly keyboard.
	PIN TextInputType = "PIN"
)

// WithDuration lets you specify a duration for how long
// a TextInput is displayed.  The default value is 0 which
// means you need to either manually clear it,.
func WithDuration(dur time.Duration) TextInputOption {
	return func(args map[string]interface{}) {
		args["Duration"] = dur.Seconds()
	}
}

// WithInputText lets you specify the text description of what the
// input box is for.
func WithInputText(text string) TextInputOption {
	return func(args map[string]interface{}) {
		args["InputText"] = text
	}
}

// WithInputType what type of input box are we looking for.
func WithInputType(inputType TextInputType) TextInputOption {
	return func(args map[string]interface{}) {
		args["InputType"] = inputType
	}
}

// WithInputKeyboardHidden lets you hide the keyboard when the
// TextInput dialog is on the screen.
func WithInputKeyboardHidden() TextInputOption {
	return func(args map[string]interface{}) {
		args["KeyboardState"] = "Closed"
	}
}

// WithPlaceholderText text is text that populates the box.  When you
// start intputing text it will clear this.
func WithPlaceholderText(text string) TextInputOption {
	return func(args map[string]interface{}) {
		args["Placeholder"] = text
	}
}

// WithSubmitText this sets the default text that lets you submit.  You
// can clear it out and add your own information.
func WithSubmitText(text string) TextInputOption {
	return func(args map[string]interface{}) {
		args["SubmitText"] = text
	}
}

// WithTitle sets the title for a TextInput.  Default is blank.
func WithTitle(text string) TextInputOption {
	return func(args map[string]interface{}) {
		args["Title"] = text
	}
}

type rpcClient interface {
	Close() error
	WriteMessage(int, *jsonrpc2.JsonRpcMessage) error
	ReadMessage() (messageType int, message *jsonrpc2.JsonRpcMessage, err error)
}

// Client is the main client that handles all communication to/from the WebEx device.
type Client struct {
	User          string
	Password      string
	Insecure      bool
	URL           string
	client        rpcClient
	seq           float64
	seqlock       sync.Mutex
	cblock        sync.Mutex
	rclock        sync.Mutex
	callbacks     map[Path]CallbackFunc
	responseChans map[float64]chan interface{}
	OnConnectFunc func(*Client)
}

// Connect to the Webex device.
func (c *Client) Connect() error {
	return c.ConnectContext(context.Background())
}

// ConnectContext connect to the Webex device with a context.
func (c *Client) ConnectContext(ctx context.Context) error {
	wsd := &websocket.Dialer{}
	wsd.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: c.Insecure,
	}

	c.callbacks = make(map[Path]CallbackFunc)
	c.responseChans = make(map[float64]chan interface{})

	encpw, err := encCreds(c.User, c.Password)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	header := http.Header{}
	header.Add(credHeader, encpw)

	wsc, hr, err := wsd.DialContext(ctx, c.URL, header)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	err = hr.Body.Close()
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	c.client = &jsonrpc2ws.Client{Conn: wsc}

	if c.OnConnectFunc != nil {
		go c.OnConnectFunc(c)
	}

	return nil
}

// Run is the client's main run loop.  This blocks till disconnect
// or a non recoverable error happens.
func (c *Client) Run() error {
	if c.client == nil {
		return ErrNotConnected
	}

	for {
		if err := c.runLoop(); err != nil {
			return err
		}
	}
}

func (c *Client) runLoop() error {
	_, msg, err := c.client.ReadMessage()
	if err != nil {
		return fmt.Errorf("runloop: %w", err)
	}

	switch msg.GetType() {
	case jsonrpc2.TypeRequestMsg:
		return fmt.Errorf("type request: %w", ErrUnsupportedMsg)

	case jsonrpc2.TypeErrorMsg:
		if err := c.chanResponse(msg, JSONRPCError{
			Code:    float64(msg.Error.Code),
			Message: msg.Error.Message,
			Data:    msg.Error.Data,
		}); err != nil {
			return err
		}

	case jsonrpc2.TypeInvalidMsg:
		if err := c.chanResponse(msg, ErrInvalidMsg); err != nil {
			return err
		}

	case jsonrpc2.TypeSuccessMsg:
		var res interface{}

		err := json.Unmarshal([]byte(*msg.Result), &res)
		if err != nil {
			return err
		}

		if err := c.chanResponse(msg, res); err != nil {
			return err
		}

	case jsonrpc2.TypeNotificationMsg:
		if err := c.runCallbacks(msg); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) runCallbacks(msg *jsonrpc2.JsonRpcMessage) error {
	event, err := oj.ParseString(string(*msg.Params))
	if err != nil {
		return fmt.Errorf("running callback: %w", err)
	}

	var (
		cbFunc CallbackFunc
		res    []interface{}
	)

	c.cblock.Lock()
	for k, v := range c.callbacks {
		cjp, err := jp.ParseString(k.toJSONPath())
		if err != nil {
			c.cblock.Unlock()

			return fmt.Errorf("jpath: %w", err)
		}

		r := cjp.Get(event)

		if len(r) > 0 && v != nil {
			res = r
			cbFunc = v

			break
		}
	}
	c.cblock.Unlock()

	if res == nil {
		return ErrMissingData
	}

	if cbFunc == nil {
		return ErrMissingCallback
	}

	go cbFunc(res)

	return nil
}

// ConnectAndRun is a helper to connect and start the run loop.
func (c *Client) ConnectAndRun() error {
	if err := c.Connect(); err != nil {
		return err
	}

	return c.Run()
}

// Close and disconnect from the Webex.
func (c *Client) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("xapi client close: %w", err)
	}

	return nil
}

// Alert displays an Alert in the UI of the device, this shows up in the upper right corner on a Desk Pro.
func (c *Client) Alert(title string, text string, duration time.Duration) error {
	args := map[string]interface{}{
		titleField:    title,
		textField:     text,
		durationField: duration.Seconds(),
	}

	_, err := c.sendCommand(alertCommand, args)

	return err
}

// TextLine displays text centered on the screen.  There is no way to dismiss this from
// the UI and requires the timeout to be a non zero value, or to be cleared with a call to TextLineClear.
func (c *Client) TextLine(text string, duration time.Duration) error {
	args := map[string]interface{}{
		textField:     text,
		durationField: duration.Seconds(),
	}

	_, err := c.sendCommand(textLineCommand, args)

	return err
}

// Prompt displays a UI prompt with multiple options.
// TODO Duration and title are optional.
// TODO we should also have a callback for the UI prompt going away
// with a timeout.
func (c *Client) Prompt(title string, text string,
	options *[optionNumberOfChoices]string, cb func(string, error)) error {
	args := map[string]interface{}{
		"FeedbackId": "go-prompt-id",
		titleField:   title,
		textField:    text,
	}

	for i, v := range options {
		args[fmt.Sprintf("Option.%d", i+1)] = v
	}

	if _, err := c.Subscribe(EventUserInterfacePromptResponse, func(data []interface{}) {
		x := data[0].(map[string]interface{})["OptionId"].(int64)

		x--

		err := c.cancelFunc(EventUserInterfacePromptResponse)()

		cb(options[int(x)], err)
	}); err != nil {
		if cerr := c.cancelFunc(EventUserInterfacePromptResponse)(); cerr != nil {
			err = multierror.Append(err, cerr)
		}

		return err
	}

	_, err := c.sendCommand(promptCommand, args)

	return err
}

// SetWidgetValue updates a UI widget with a new value.
func (c *Client) SetWidgetValue(widgetID string, value interface{}) error {
	_, err := c.sendCommand(widgetSetValueCommand, map[string]interface{}{
		"WidgetId": widgetID,
		"Value":    value,
	})

	return err
}

// TextInput lets you prompt a user for a free form text string.  When the user submits
// the text the callback is called with the response passed in.
func (c *Client) TextInput(text string, cb func(canceled bool, response string, err error), opts ...TextInputOption) error {
	args := map[string]interface{}{
		"FeedbackId": "go-text-input-id",
		textField:    text,
	}

	for _, v := range opts {
		v(args)
	}

	canFunc := func() error {
		var res error
		if err := c.cancelFunc(EventUserInterfaceTextInputResponse)(); err != nil {
			res = multierror.Append(res, err)
		}

		if err := c.cancelFunc(EventUserInterfaceTextInputResponseClear)(); err != nil {
			res = multierror.Append(res, err)
		}

		return res
	}

	if _, err := c.Subscribe(EventUserInterfaceTextInputResponse,
		func(data []interface{}) {
			err := canFunc()
			cb(false, data[0].(map[string]interface{})[textField].(string), err)
		}); err != nil {
		return multierror.Append(err, canFunc())
	}

	if _, err := c.Subscribe(EventUserInterfaceTextInputResponseClear,
		func(data []interface{}) {
			err := canFunc()
			cb(true, "", err)
		}); err != nil {
		return multierror.Append(err, canFunc())
	}

	_, err := c.sendCommand(textInputCommand, args)

	return err
}

// Rating opens a '5 star' rating dialog on the Webex device.  A user can
// cancel the prompt or choose a number of stars.  The rating is returned
// as an int64.
func (c *Client) Rating(title string, text string, callback func(canceled bool, value int64, err error)) error {
	args := map[string]interface{}{
		"FeedbackId": "go-rating-id",
		titleField:   title,
		textField:    text,
	}

	cleanup := func() error {
		var res error
		if err := c.cancelFunc(EventUserInterfaceRatingResponse)(); err != nil {
			res = multierror.Append(res, err)
		}

		if err := c.cancelFunc(EventUserInterfaceMessageRatingCleared)(); err != nil {
			res = multierror.Append(res, err)
		}

		return res
	}

	if _, err := c.Subscribe(EventUserInterfaceRatingResponse,
		func(data []interface{}) {
			x := data[0].(map[string]interface{})["Rating"].(int64)

			callback(false, x, cleanup())
		}); err != nil {
		return multierror.Append(err, cleanup())
	}

	if _, err := c.Subscribe(EventUserInterfaceMessageRatingCleared,
		func(data []interface{}) {
			callback(true, 0, cleanup())
		}); err != nil {
		return multierror.Append(err, cleanup())
	}

	_, err := c.sendCommand(ratingCommand, args)

	return err
}

// Subscribe lets you subscribe to event, UI or status change events of the Webex device.
func (c *Client) Subscribe(path Path, callback CallbackFunc) (func() error, error) {
	_, err := c.sendCommand(feedbackSusbscribe, path.toSubQuery())
	if err != nil {
		return nil, err
	}

	c.cblock.Lock()
	defer c.cblock.Unlock()
	c.callbacks[path] = callback

	return c.cancelFunc(path), nil
}

// Get retrieve the value of a setting, status or UI element.
func (c *Client) Get(path Path) (interface{}, error) {
	return c.sendCommand(getCommand, path.toGetParams())
}

func (c *Client) Mute() error {
	_, err := c.sendCommand(muteCommand, nil)
	return err
}

func (c *Client) UnMute() error {
	_, err := c.sendCommand(unmuteCommand, nil)
	return err
}

func (c *Client) chanResponse(msg *jsonrpc2.JsonRpcMessage, res interface{}) error {
	k, ok := msg.ID.(float64)
	if !ok {
		return ErrMissingIDField
	}

	c.rclock.Lock()
	ch, ok := c.responseChans[k]
	c.rclock.Unlock()

	if !ok {
		return ErrMissingChannel
	}

	ch <- res

	return nil
}

func (c *Client) cancelFunc(path Path) func() error {
	return func() error {
		c.cblock.Lock()
		defer c.cblock.Unlock()
		delete(c.callbacks, path)

		_, err := c.sendCommand(feedbackUnsubscribe, path.toSubQuery())

		return err
	}
}

func (c *Client) sendCommand(command Command, params interface{}) (interface{}, error) {
	if c.client == nil {
		return nil, ErrNotConnected
	}

	c.seqlock.Lock()
	c.seq++
	myseq := c.seq
	c.seqlock.Unlock()

	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	msg := jsonrpc2.NewJsonRpcRequest(myseq, string(command), data)
	rc := make(chan interface{})

	defer func() {
		c.rclock.Lock()
		delete(c.responseChans, myseq)
		c.rclock.Unlock()
		close(rc)
	}()

	c.rclock.Lock()
	c.responseChans[myseq] = rc
	c.rclock.Unlock()

	err = c.client.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return nil, fmt.Errorf("write message: %w", err)
	}

	r := <-rc

	switch v := r.(type) {
	case error:
		return nil, r.(error)
	case map[string]interface{}, float64:
		return r, nil
	default:
		return nil, fmt.Errorf("receive: %+V, %w", v, ErrUnknownResponse)
	}
}

func encCreds(user string, password string) (string, error) {
	if user == "" || password == "" {
		return "", ErrInvalidCredentials
	}

	re := strings.NewReplacer("+", "-", "/", "_", "=", "")

	return credPrefix + re.Replace(base64.StdEncoding.EncodeToString([]byte(user+":"+password))), nil
}
