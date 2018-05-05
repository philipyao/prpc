package client

type configSelect struct {
	typ   selectType
	index int //specify which endpoint to select
}

//service 相关option
type fnOptionService func(sc *svcClient) error

func WithVersion(version string) fnOptionService {
	//指定匹配特定版本
	return func(sc *svcClient) error {
		sc.setVersion(version)
		return nil
	}
}
func WithVersionAll() fnOptionService {
	//要兼容所有版本
	return WithVersion(noSpecifiedVersion)
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
