package apps

// AppWithStatus extends AppEntry with runtime status
type AppWithStatus struct {
	AppEntry
	Status        string
	LocalVersion  string
	RemoteVersion string
}
