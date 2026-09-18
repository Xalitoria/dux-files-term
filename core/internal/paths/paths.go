package paths

import dankpaths "github.com/AvengeMedia/dankgo/paths"

var app = dankpaths.New("dfiles")

func ConfigDir() (string, error) { return app.ConfigDir() }

func DataDir() (string, error) { return app.DataDir() }

func CacheDir() (string, error) { return app.CacheDir() }

func StateDir() (string, error) { return app.StateDir() }

func SocketDir() string { return app.SocketDir() }

func SocketPath() string { return app.SocketPath() }
