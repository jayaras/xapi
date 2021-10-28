package xapi

import (
	"fmt"
	"strings"
)

// Path represents the xapi path in the system.  It is a space delimited field that can be converted
// to jpath, parameters for a Get or the path to listen to for events.  If the needed Path for your
// use case is not implemented here its possible to just define a new path and use all the library
// functionality with that new path.
type Path string

func (p Path) toSubQuery() map[string]interface{} {
	return map[string]interface{}{
		"Query": strings.Fields(string(p)),
	}
}

func (p Path) toGetParams() map[string]interface{} {
	return map[string]interface{}{
		"Path": strings.Fields(string(p)),
	}
}

func (p Path) toJSONPath() string {
	result := "$."
	for _, x := range strings.Fields(string(p)) {
		result = fmt.Sprintf("%s.%s", result, x)
	}

	return result
}

const (
	Status                                    Path = "Status"
	StatusSystemUnit                          Path = "Status SystemUnit"
	StatusSystemUnitStateNumberOfActiveCalls  Path = "Status SystemUnit State NumberOfActiveCalls"
	StatusAudioVolumeLevel                    Path = "Status Audio Volume"
	StatusAudioMicrophonesMute                Path = "Status Audio Microphones Mute"
	StatusVideoInputMainVideoMute             Path = "Status Video Input MainVideoMute"
	Event                                     Path = "Event"
	EventUserInterface                        Path = "Event UserInterface"
	EventUserInterfaceExtension               Path = "Event UserInterface Extensions"
	EventUserInterfaceExtensionsEvent         Path = "Event UserInterface Extensions Event"
	EventUserInterfaceExtensionsEventReleased Path = "Event UserInterface Extensions Event Released"
	EventUserInterfaceExtensionsEventClicked  Path = "Event UserInterface Extensions Event Pressed"
	EventUserInterfaceExtensionsEventChanged  Path = "Event UserInterface Extensions Event Changed"
	EventUserInterfaceWidgetAction            Path = "Event UserInterface Extensions Widget Action"
	EventUserInterfacePanelClicked            Path = "Event UserInterface Extensions Panel Clicked"
	EventUserInterfacePanelClose              Path = "Event UserInterface Extensions Panel Close"
	EventUserInterfacePanelOpen               Path = "Event UserInterface Extensions Panel Open"
	EventUserInterfacePromptResponse          Path = "Event UserInterface Message Prompt Response"
	EventUserInterfaceRatingResponse          Path = "Event UserInterface Message Rating Response"
	EventUserInterfaceTextInputResponse       Path = "Event UserInterface Message TextInput Response"
	EventUserInterfaceTextInputResponseClear  Path = "Event UserInterface Message TextInput Clear"
	EventUserInterfaceMessageAlertCleared     Path = "Event UserInterface Message Alert Cleared"
	EventUserInterfaceMessageRatingCleared    Path = "Event UserInterface Message Rating Cleared"
	EventUserInterfaceMessageTextLineCleared  Path = "Event UserInterface Message TextLine Cleared"
	EventShutdown                             Path = "Event Shutdown"
	EventIncomingCallIndication               Path = "Event IncomingCallIndication"
)
