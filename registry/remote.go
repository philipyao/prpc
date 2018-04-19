package registry

type remote interface {
    Connect()
    Create(path string, empehter bool)
    Close()
    Watch(path string)
    WatchChildren(path string)
}
