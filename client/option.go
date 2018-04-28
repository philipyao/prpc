package client

type configSelect struct {
	typ 	selectType
	index 	int		//specify which endpoints to select
}

//service 相关option
type fnOptionService func(sc *svcClient) error
func WithVersion(version int) fnOptionService {
	return func(sc *svcClient) error {
		sc.setVersion(version)
		return nil
	}
}
func WithIndex(index int) fnOptionService {
	return func(sc *svcClient) error {
		return sc.setIndex(index)
	}
}
func WithSelectType(styp selectType) fnOptionService {
	return func(sc *svcClient) error {
		return sc.setSelectType(styp)
	}
}
