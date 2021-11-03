Background
===
 I was issued a Webex Desk Pro as part of my day job. The first thing I wanted  was use its customization/integreation support to Home Assistant so I could adjust my blinds and lighting as the day chanes right from my Desk Pro.  The out of the box integreation options are JavaScript or Python SDKs and I played around with them but in the end Golang was the best fit for me.  

Known Limitations
===
- Not all event and command types are currently implemented.  Only enough for most of my current use cases.
- I only have access to a Desk Pro with integrator level access.
- This code is still a WIP so the API should __NOT__ be considered stable.
- WebSocket only.  No SSH transport.


Todo
===
- Unit Tests
- Better/More Examples
- Improve Docs
- Review that all optional fields are infact optional for things like `Client.Alert`.


