package main

import "github.com/lucian95511/and/internal/pluginapi"

// istemci, DeltaX'in AND'ın eklenti API'sinden kullandığı kısımdır.
// *pluginapi.Client bunu karşılar; testler sahtesini kullanır.
type istemci interface {
	PollDM() ([]pluginapi.DMMsg, error)
	DMHistoryFrom(id string) ([]pluginapi.DMRecord, error)
	SendDM(peerID, message string) error
	SendDMQueued(peerID, message string) error
	SendFile(peerID, localPath string) error
	PollFile() ([]pluginapi.FileMsg, error)
	PollConsent() ([]pluginapi.FileConsentReq, error)
	RespondConsent(transferID string, accept bool) error
	Peers() ([]pluginapi.PeerInfo, error)
}
