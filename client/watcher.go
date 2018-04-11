package client

import (
    "github.com/philipyao/toolbox/zkcli"
)

type watcher struct {
    conn *zkcli.Conn
    path string
    stop chan struct{}
}

func newWatcher(conn *zkcli.Conn, path string) *watcher {
    return &watcher{
        conn: conn,
        path: path,
        stop: make(chan struct{}, 1),
    }
}

func (w *watcher) Watch(cb func(p string, c []string, e error)) error {
    return w.conn.WatchChildren(w.path, cb, w.stop)
}

func (w *watcher) Stop() {
    w.stop <- struct{}{}
}