// Package registry is an interface for service discovery
package registry

type Registry struct {
    rt remote
    fb failback
    c cache
}

func (r *Registry) Register() {

}

func (r *Registry) Subscribe() {

}

func (r *Registry) Lookup() {

}
